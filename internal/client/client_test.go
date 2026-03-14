package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jv/twenty-crm-cli/internal/config"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAuthCheckUsesExpectedEndpointAndAuthHeader(t *testing.T) {
	cfg, err := config.New("secret", "https://api.twenty.com", "json")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var gotPath string
	var gotAuth string
	var gotContentType string

	cli := New(cfg, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotPath = req.URL.String()
		gotAuth = req.Header.Get("Authorization")
		gotContentType = req.Header.Get("Content-Type")

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":{"__typename":"Query"}}`)),
		}, nil
	}))

	result, err := cli.AuthCheck(context.Background())
	if err != nil {
		t.Fatalf("AuthCheck() error = %v", err)
	}

	if gotPath != "https://api.twenty.com/metadata" {
		t.Fatalf("path = %q", gotPath)
	}

	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}

	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}

	if result.Endpoint != "/metadata" {
		t.Fatalf("Endpoint = %q", result.Endpoint)
	}
}

func TestMetadataObjectsUsesMetadataEndpointAndParsesObjects(t *testing.T) {
	cfg, err := config.New("secret", "https://api.twenty.com", "json")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var gotPath string
	var gotAuth string
	var gotContentType string

	cli := New(cfg, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotPath = req.URL.String()
		gotAuth = req.Header.Get("Authorization")
		gotContentType = req.Header.Get("Content-Type")

		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"objects": {
						"edges": [
							{
								"node": {
									"id": "obj_1",
									"universalIdentifier": "u_person",
									"nameSingular": "person",
									"namePlural": "people",
									"labelSingular": "Person",
									"labelPlural": "People",
									"description": "Contacts",
									"isActive": true,
									"labelIdentifierFieldMetadataId": "fld_1",
									"imageIdentifierFieldMetadataId": "fld_2",
									"fieldsList": [
										{
											"id": "fld_1",
											"universalIdentifier": "u_email",
											"type": "EMAIL",
											"name": "email",
											"label": "Email",
											"description": "Primary email",
											"isActive": true,
											"relation": null
										}
									]
								}
							}
						]
					}
				}
			}`)),
		}, nil
	}))

	objects, err := cli.MetadataObjects(context.Background())
	if err != nil {
		t.Fatalf("MetadataObjects() error = %v", err)
	}

	if gotPath != "https://api.twenty.com/metadata" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if len(objects) != 1 {
		t.Fatalf("len(objects) = %d, want 1", len(objects))
	}
	if objects[0].NameSingular != "person" {
		t.Fatalf("object name = %q", objects[0].NameSingular)
	}
	if len(objects[0].Fields) != 1 || objects[0].Fields[0].Name != "email" {
		t.Fatalf("fields = %#v", objects[0].Fields)
	}
}
