package codepush

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ErrDuplicateRelease is returned by Promote when the target deployment already
// contains a release with identical content. Use errors.Is to detect it and
// implement --no-duplicate-release-error behaviour.
//
// NOTE: detection relies on the server's current error message text. If the
// server team changes the message in internal/service/promote.go, this
// detection will silently stop working and --no-duplicate-release-error will
// behave like a normal error again. Update both sides when the server changes.
var ErrDuplicateRelease = errors.New("duplicate release")

// HTTPClient implements Client using net/http.
type HTTPClient struct {
	BaseURL string
	Token   string
	version string
	client  *http.Client
}

// NewHTTPClient creates a new HTTPClient.
func NewHTTPClient(baseURL, token, version string) *HTTPClient {
	if version == "" {
		version = "unknown"
	}
	return &HTTPClient{
		BaseURL: baseURL,
		Token:   token,
		version: version,
		client:  &http.Client{},
	}
}

// buildURL assembles a relative URL from a path and optional query parameters.
// Path parameters must be escaped with url.PathEscape before being interpolated
// into path. Query values are safely encoded by url.Values.Encode.
func buildURL(path string, params url.Values) string {
	if len(params) == 0 {
		return path
	}
	return path + "?" + params.Encode()
}

// ListDeployments returns all deployments for the given app.
func (c *HTTPClient) ListDeployments(ctx context.Context, appID string) ([]Deployment, error) {
	params := url.Values{}
	params.Set("app_id", appID)
	path := buildURL("/deployments", params)

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result DeploymentListResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}

	return result.Items, nil
}

// CreateDeployment creates a new deployment.
func (c *HTTPClient) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) (*Deployment, error) {
	resp, err := c.doJSONRequest(ctx, http.MethodPost, "/deployments", req)
	if err != nil {
		return nil, err
	}

	var result Deployment
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("creating deployment: %w", err)
	}

	return &result, nil
}

// GetDeployment returns a single deployment by ID.
func (c *HTTPClient) GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error) {
	path := fmt.Sprintf("/deployments/%s", url.PathEscape(deploymentID))

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result Deployment
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("getting deployment: %w", err)
	}

	return &result, nil
}

// RenameDeployment renames an existing deployment.
func (c *HTTPClient) RenameDeployment(ctx context.Context, deploymentID string, req RenameDeploymentRequest) (*Deployment, error) {
	path := fmt.Sprintf("/deployments/%s", url.PathEscape(deploymentID))

	resp, err := c.doJSONRequest(ctx, http.MethodPatch, path, req)
	if err != nil {
		return nil, err
	}

	var result Deployment
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("renaming deployment: %w", err)
	}

	return &result, nil
}

// DeleteDeployment deletes a deployment.
func (c *HTTPClient) DeleteDeployment(ctx context.Context, deploymentID string) error {
	path := fmt.Sprintf("/deployments/%s", url.PathEscape(deploymentID))

	resp, err := c.doRequest(ctx, http.MethodDelete, path)
	if err != nil {
		return err
	}

	if err := decodeResponse(resp, nil); err != nil {
		return fmt.Errorf("deleting deployment: %w", err)
	}

	return nil
}

// GetUploadURL requests a signed upload URL for a new update.
func (c *HTTPClient) GetUploadURL(ctx context.Context, deploymentID, updateID string, req UploadURLRequest) (*UploadURLResponse, error) {
	path := fmt.Sprintf("/updates/%s/upload-url", url.PathEscape(updateID))

	params := url.Values{}
	params.Set("deployment_id", deploymentID)
	params.Set("app_version", req.AppVersion)
	params.Set("file_name", req.FileName)
	params.Set("file_size_bytes", strconv.FormatInt(req.FileSizeBytes, 10))
	if req.Description != "" {
		params.Set("description", req.Description)
	}
	if req.Mandatory {
		params.Set("mandatory", "true")
	}
	if req.Disabled {
		params.Set("disabled", "true")
	}
	if req.Rollout != nil {
		params.Set("rollout", strconv.FormatFloat(*req.Rollout, 'f', -1, 64))
	}

	resp, err := c.doRequest(ctx, http.MethodGet, buildURL(path, params))
	if err != nil {
		return nil, err
	}

	var result UploadURLResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("getting upload URL: %w", err)
	}

	return &result, nil
}

// UploadFile uploads the zip file to the signed URL.
func (c *HTTPClient) UploadFile(ctx context.Context, ufr UploadFileRequest) error {
	req, err := http.NewRequestWithContext(ctx, ufr.Method, ufr.URL, ufr.Body)
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}

	req.ContentLength = ufr.ContentLength
	for k, v := range ufr.Headers {
		req.Header.Set(k, v)
	}
	// Set after upload headers so CLI identity is always authoritative.
	req.Header.Set("X-Bitrise-User-Agent", "codepush-cli/"+c.version)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetUpdateStatus polls the status of an update.
func (c *HTTPClient) GetUpdateStatus(ctx context.Context, updateID string) (*UpdateStatus, error) {
	path := fmt.Sprintf("/updates/%s/status", url.PathEscape(updateID))

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result UpdateStatus
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("getting update status: %w", err)
	}

	return &result, nil
}

// ListUpdates returns all updates for a deployment.
func (c *HTTPClient) ListUpdates(ctx context.Context, deploymentID string) ([]Update, error) {
	params := url.Values{}
	params.Set("deployment_id", deploymentID)
	path := buildURL("/updates", params)

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result UpdateListResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("listing updates: %w", err)
	}

	return result.Items, nil
}

// GetUpdate returns a single update by ID.
func (c *HTTPClient) GetUpdate(ctx context.Context, updateID string) (*Update, error) {
	path := fmt.Sprintf("/updates/%s", url.PathEscape(updateID))

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result Update
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("getting update: %w", err)
	}

	return &result, nil
}

// PatchUpdate updates metadata on an existing update.
func (c *HTTPClient) PatchUpdate(ctx context.Context, updateID string, req PatchRequest) (*Update, error) {
	path := fmt.Sprintf("/updates/%s", url.PathEscape(updateID))

	resp, err := c.doJSONRequest(ctx, http.MethodPatch, path, req)
	if err != nil {
		return nil, err
	}

	var result Update
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("patching update: %w", err)
	}

	return &result, nil
}

// DeleteUpdate deletes an update.
func (c *HTTPClient) DeleteUpdate(ctx context.Context, updateID string) error {
	path := fmt.Sprintf("/updates/%s", url.PathEscape(updateID))

	resp, err := c.doRequest(ctx, http.MethodDelete, path)
	if err != nil {
		return err
	}

	if err := decodeResponse(resp, nil); err != nil {
		return fmt.Errorf("deleting update: %w", err)
	}

	return nil
}

// Rollback sends a rollback request for a deployment.
func (c *HTTPClient) Rollback(ctx context.Context, deploymentID string, req RollbackRequest) (*Update, error) {
	path := fmt.Sprintf("/deployments/%s/rollback", url.PathEscape(deploymentID))

	resp, err := c.doJSONRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	var result Update
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("rolling back deployment: %w", err)
	}

	return &result, nil
}

// Promote sends a promote request for a deployment.
// Returns ErrDuplicateRelease (wrapped) when the server rejects the request
// because the target deployment already contains identical content.
func (c *HTTPClient) Promote(ctx context.Context, deploymentID string, req PromoteRequest) (*Update, error) {
	path := fmt.Sprintf("/deployments/%s/promote", url.PathEscape(deploymentID))

	resp, err := c.doJSONRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if strings.Contains(string(body), "ERR_BAD_REQUEST") && strings.Contains(string(body), "identical to the contents") {
			return nil, fmt.Errorf("promoting deployment: %w", ErrDuplicateRelease)
		}
		return nil, fmt.Errorf("promoting deployment: API returned HTTP 400: %s", string(body))
	}

	var result Update
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("promoting deployment: %w", err)
	}

	return &result, nil
}

func (c *HTTPClient) doJSONRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Bitrise-User-Agent", "codepush-cli/"+c.version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request to %s: %w", path, err)
	}

	return resp, nil
}

func (c *HTTPClient) doRequest(ctx context.Context, method, path string) (*http.Response, error) {
	reqURL := c.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Bitrise-User-Agent", "codepush-cli/"+c.version)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request to %s: %w", path, err)
	}

	return resp, nil
}

func decodeResponse(resp *http.Response, v any) error {
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
