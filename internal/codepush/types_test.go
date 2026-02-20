package codepush

import (
	"encoding/json"
	"testing"
)

func TestHeaderMap_UnmarshalJSON_Object(t *testing.T) {
	data := `{"url":"https://example.com","method":"PUT","headers":{"Content-Type":"application/zip","x-custom":"value"}}`

	var resp UploadURLResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Headers["Content-Type"] != "application/zip" {
		t.Errorf("Content-Type: got %q, want %q", resp.Headers["Content-Type"], "application/zip")
	}
	if resp.Headers["x-custom"] != "value" {
		t.Errorf("x-custom: got %q, want %q", resp.Headers["x-custom"], "value")
	}
}

func TestHeaderMap_UnmarshalJSON_Array(t *testing.T) {
	data := `{"url":"https://example.com","method":"PUT","headers":[{"key":"Content-Type","value":"application/zip"},{"key":"x-custom","value":"value"}]}`

	var resp UploadURLResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Headers["Content-Type"] != "application/zip" {
		t.Errorf("Content-Type: got %q, want %q", resp.Headers["Content-Type"], "application/zip")
	}
	if resp.Headers["x-custom"] != "value" {
		t.Errorf("x-custom: got %q, want %q", resp.Headers["x-custom"], "value")
	}
}

func TestHeaderMap_UnmarshalJSON_Null(t *testing.T) {
	data := `{"url":"https://example.com","method":"PUT","headers":null}`

	var resp UploadURLResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Headers != nil {
		t.Errorf("expected nil headers, got %v", resp.Headers)
	}
}

func TestHeaderMap_UnmarshalJSON_EmptyArray(t *testing.T) {
	data := `{"url":"https://example.com","method":"PUT","headers":[]}`

	var resp UploadURLResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Headers) != 0 {
		t.Errorf("expected empty headers, got %v", resp.Headers)
	}
}

func TestHeaderMap_UnmarshalJSON_ArrayWithName(t *testing.T) {
	data := `{"url":"https://example.com","method":"PUT","headers":[{"name":"Content-Type","value":"application/zip"},{"name":"x-custom","value":"value"}]}`

	var resp UploadURLResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Headers["Content-Type"] != "application/zip" {
		t.Errorf("Content-Type: got %q, want %q", resp.Headers["Content-Type"], "application/zip")
	}
	if resp.Headers["x-custom"] != "value" {
		t.Errorf("x-custom: got %q, want %q", resp.Headers["x-custom"], "value")
	}
}

func TestHeaderMap_UnmarshalJSON_SkipsEmptyKeys(t *testing.T) {
	data := `{"url":"https://example.com","method":"PUT","headers":[{"key":"Content-Type","value":"application/zip"},{"key":"","value":"orphan"}]}`

	var resp UploadURLResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Headers) != 1 {
		t.Errorf("expected 1 header, got %d: %v", len(resp.Headers), resp.Headers)
	}
	if resp.Headers["Content-Type"] != "application/zip" {
		t.Errorf("Content-Type: got %q, want %q", resp.Headers["Content-Type"], "application/zip")
	}
}

func TestHeaderMap_UnmarshalJSON_Invalid(t *testing.T) {
	data := `{"url":"https://example.com","method":"PUT","headers":"bad"}`

	var resp UploadURLResponse
	if err := json.Unmarshal([]byte(data), &resp); err == nil {
		t.Fatal("expected error for invalid headers format")
	}
}
