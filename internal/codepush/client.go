package codepush

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// HTTPClient implements Client using net/http.
type HTTPClient struct {
	BaseURL string
	Token   string
	client  *http.Client
}

// NewHTTPClient creates a new HTTPClient.
func NewHTTPClient(baseURL, token string) *HTTPClient {
	return &HTTPClient{
		BaseURL: baseURL,
		Token:   token,
		client:  &http.Client{},
	}
}

// ListDeployments returns all deployments for the release management app.
func (c *HTTPClient) ListDeployments(ctx context.Context, appID string) ([]Deployment, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments", appID)

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
func (c *HTTPClient) CreateDeployment(ctx context.Context, appID string, req CreateDeploymentRequest) (*Deployment, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments", appID)

	resp, err := c.doJSONRequest(ctx, http.MethodPost, path, req)
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
func (c *HTTPClient) GetDeployment(ctx context.Context, appID, deploymentID string) (*Deployment, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s", appID, deploymentID)

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
func (c *HTTPClient) RenameDeployment(ctx context.Context, appID, deploymentID string, req RenameDeploymentRequest) (*Deployment, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s", appID, deploymentID)

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
func (c *HTTPClient) DeleteDeployment(ctx context.Context, appID, deploymentID string) error {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s", appID, deploymentID)

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
func (c *HTTPClient) GetUploadURL(ctx context.Context, appID, deploymentID, updateID string, req UploadURLRequest) (*UploadURLResponse, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s/upload-url",
		appID, deploymentID, updateID)

	params := url.Values{}
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
	if req.Rollout > 0 && req.Rollout < 100 {
		params.Set("rollout", strconv.Itoa(req.Rollout))
	}

	fullPath := path + "?" + params.Encode()

	resp, err := c.doRequest(ctx, http.MethodGet, fullPath)
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
func (c *HTTPClient) GetUpdateStatus(ctx context.Context, appID, deploymentID, updateID string) (*UpdateStatus, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s/status",
		appID, deploymentID, updateID)

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
func (c *HTTPClient) ListUpdates(ctx context.Context, appID, deploymentID string) ([]Update, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages", appID, deploymentID)

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
func (c *HTTPClient) GetUpdate(ctx context.Context, appID, deploymentID, updateID string) (*Update, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s",
		appID, deploymentID, updateID)

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
func (c *HTTPClient) PatchUpdate(ctx context.Context, appID, deploymentID, updateID string, req PatchRequest) (*Update, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s",
		appID, deploymentID, updateID)

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

// DeleteUpdate deletes an update from a deployment.
func (c *HTTPClient) DeleteUpdate(ctx context.Context, appID, deploymentID, updateID string) error {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s",
		appID, deploymentID, updateID)

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
func (c *HTTPClient) Rollback(ctx context.Context, appID, deploymentID string, req RollbackRequest) (*Update, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/rollback", appID, deploymentID)

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
func (c *HTTPClient) Promote(ctx context.Context, appID, deploymentID string, req PromoteRequest) (*Update, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/promote", appID, deploymentID)

	resp, err := c.doJSONRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
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
