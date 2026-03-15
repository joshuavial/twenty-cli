package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type PageInfo struct {
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}

type ListResult struct {
	Records    []map[string]any
	TotalCount int
	PageInfo   PageInfo
	Endpoint   string
}

type RecordResult struct {
	Record   map[string]any
	Endpoint string
}

func (c *Client) ListRecords(ctx context.Context, plural string, query url.Values) (ListResult, error) {
	endpoint := c.baseURL + "/rest/" + plural
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	resp, err := c.doJSONRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ListResult{}, err
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp); err != nil {
		return ListResult{}, err
	}

	var payload struct {
		Data      map[string][]map[string]any `json:"data"`
		TotalCount int                        `json:"totalCount"`
		PageInfo  PageInfo                    `json:"pageInfo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Records:    payload.Data[plural],
		TotalCount: payload.TotalCount,
		PageInfo:   payload.PageInfo,
		Endpoint:   sanitizeEndpoint(endpoint),
	}, nil
}

func (c *Client) GetRecord(ctx context.Context, plural, singular, id string, query url.Values) (RecordResult, error) {
	endpoint := c.baseURL + "/rest/" + plural + "/" + id
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	resp, err := c.doJSONRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return RecordResult{}, err
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp); err != nil {
		return RecordResult{}, err
	}

	record, err := decodeRecordBody(resp.Body, singular)
	if err != nil {
		return RecordResult{}, err
	}

	return RecordResult{
		Record:   record,
		Endpoint: sanitizeEndpoint(endpoint),
	}, nil
}

func (c *Client) CreateRecord(ctx context.Context, plural, action string, payload map[string]any) (RecordResult, error) {
	endpoint := c.baseURL + "/rest/" + plural

	resp, err := c.doJSONRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return RecordResult{}, err
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp); err != nil {
		return RecordResult{}, err
	}

	record, err := decodeRecordBody(resp.Body, action)
	if err != nil {
		return RecordResult{}, err
	}

	return RecordResult{
		Record:   record,
		Endpoint: sanitizeEndpoint(endpoint),
	}, nil
}

func (c *Client) UpdateRecord(ctx context.Context, plural, action, id string, payload map[string]any) (RecordResult, error) {
	endpoint := c.baseURL + "/rest/" + plural + "/" + id

	resp, err := c.doJSONRequest(ctx, http.MethodPatch, endpoint, payload)
	if err != nil {
		return RecordResult{}, err
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp); err != nil {
		return RecordResult{}, err
	}

	record, err := decodeRecordBody(resp.Body, action)
	if err != nil {
		return RecordResult{}, err
	}

	return RecordResult{
		Record:   record,
		Endpoint: sanitizeEndpoint(endpoint),
	}, nil
}

func decodeRecordBody(r io.Reader, key string) (map[string]any, error) {
	var payload struct {
		Data map[string]map[string]any `json:"data"`
	}
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, err
	}

	record, ok := payload.Data[key]
	if !ok {
		return nil, fmt.Errorf("response missing expected key %q in data", key)
	}

	return record, nil
}

func ensureHTTPSuccess(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return &APIError{
		StatusCode: resp.StatusCode,
		Body:       strings.TrimSpace(string(body)),
	}
}
