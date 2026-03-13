package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/jv/twenty-crm-cli/internal/config"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient HTTPDoer
}

type AuthCheckResult struct {
	StatusCode int    `json:"status_code"`
	Endpoint   string `json:"endpoint"`
}

type MetadataObject struct {
	ID                             string          `json:"id"`
	UniversalIdentifier            string          `json:"universalIdentifier"`
	NameSingular                   string          `json:"nameSingular"`
	NamePlural                     string          `json:"namePlural"`
	LabelSingular                  string          `json:"labelSingular"`
	LabelPlural                    string          `json:"labelPlural"`
	Description                    string          `json:"description"`
	IsActive                       bool            `json:"isActive"`
	LabelIdentifierFieldMetadataID string          `json:"labelIdentifierFieldMetadataId"`
	ImageIdentifierFieldMetadataID string          `json:"imageIdentifierFieldMetadataId"`
	Fields                         []MetadataField `json:"fieldsList"`
}

type MetadataField struct {
	ID                  string            `json:"id"`
	UniversalIdentifier string            `json:"universalIdentifier"`
	Type                string            `json:"type"`
	Name                string            `json:"name"`
	Label               string            `json:"label"`
	Description         string            `json:"description"`
	IsActive            bool              `json:"isActive"`
	Relation            *MetadataRelation `json:"relation"`
}

type MetadataRelation struct {
	Type         string            `json:"type"`
	SourceObject MetadataObjectRef `json:"sourceObjectMetadata"`
	TargetObject MetadataObjectRef `json:"targetObjectMetadata"`
	SourceField  MetadataFieldRef  `json:"sourceFieldMetadata"`
	TargetField  MetadataFieldRef  `json:"targetFieldMetadata"`
}

type MetadataObjectRef struct {
	ID           string `json:"id"`
	NameSingular string `json:"nameSingular"`
	NamePlural   string `json:"namePlural"`
}

type MetadataFieldRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIError struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api request failed with status %d", e.StatusCode)
}

func New(cfg config.Config, httpClient HTTPDoer) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
	}
}

func (c *Client) AuthCheck(ctx context.Context) (AuthCheckResult, error) {
	endpoint := c.baseURL + "/rest/metadata"

	resp, err := c.doJSONRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return AuthCheckResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return AuthCheckResult{}, &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	return AuthCheckResult{
		StatusCode: resp.StatusCode,
		Endpoint:   sanitizeEndpoint(endpoint),
	}, nil
}

func (c *Client) MetadataObjects(ctx context.Context) ([]MetadataObject, error) {
	const query = `
query CLIObjectMetadataItems {
  objects(paging: { first: 1000 }, filter: { isActive: { is: true } }) {
    edges {
      node {
        id
        universalIdentifier
        nameSingular
        namePlural
        labelSingular
        labelPlural
        description
        isActive
        labelIdentifierFieldMetadataId
        imageIdentifierFieldMetadataId
        fieldsList {
          id
          universalIdentifier
          type
          name
          label
          description
          isActive
          relation {
            type
            sourceObjectMetadata {
              id
              nameSingular
              namePlural
            }
            targetObjectMetadata {
              id
              nameSingular
              namePlural
            }
            sourceFieldMetadata {
              id
              name
            }
            targetFieldMetadata {
              id
              name
            }
          }
        }
      }
    }
  }
}`

	body := map[string]string{"query": query}
	resp, err := c.doJSONRequest(ctx, http.MethodPost, c.baseURL+"/metadata", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(payload)),
		}
	}

	var result struct {
		Data struct {
			Objects struct {
				Edges []struct {
					Node MetadataObject `json:"node"`
				} `json:"edges"`
			} `json:"objects"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Errors) > 0 {
		payload, _ := json.Marshal(result.Errors)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(payload),
		}
	}

	objects := make([]MetadataObject, 0, len(result.Data.Objects.Edges))
	for _, edge := range result.Data.Objects.Edges {
		objects = append(objects, edge.Node)
	}

	return objects, nil
}

func sanitizeEndpoint(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if parsed.RawQuery == "" {
		return parsed.Path
	}

	return parsed.Path + "?" + parsed.RawQuery
}

func (c *Client) doJSONRequest(ctx context.Context, method, endpoint string, body any) (*http.Response, error) {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}
