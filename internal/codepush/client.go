package codepush

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// HTTPClient implements Client using net/http.
type HTTPClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewHTTPClient creates a new HTTPClient.
func NewHTTPClient(baseURL, token string) *HTTPClient {
	return &HTTPClient{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

// ListDeployments returns all deployments for the connected app.
func (c *HTTPClient) ListDeployments(appID string) ([]Deployment, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments", appID)

	resp, err := c.doRequest(http.MethodGet, path, nil)
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
func (c *HTTPClient) CreateDeployment(appID string, req CreateDeploymentRequest) (*Deployment, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments", appID)

	resp, err := c.doJSONRequest(http.MethodPost, path, req)
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
func (c *HTTPClient) GetDeployment(appID, deploymentID string) (*Deployment, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s", appID, deploymentID)

	resp, err := c.doRequest(http.MethodGet, path, nil)
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
func (c *HTTPClient) RenameDeployment(appID, deploymentID string, req RenameDeploymentRequest) (*Deployment, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s", appID, deploymentID)

	resp, err := c.doJSONRequest(http.MethodPatch, path, req)
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
func (c *HTTPClient) DeleteDeployment(appID, deploymentID string) error {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s", appID, deploymentID)

	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	if err := decodeResponse(resp, nil); err != nil {
		return fmt.Errorf("deleting deployment: %w", err)
	}

	return nil
}

// GetUploadURL requests a signed upload URL for a new package.
func (c *HTTPClient) GetUploadURL(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s/upload-url",
		appID, deploymentID, packageID)

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

	resp, err := c.doRequest(http.MethodGet, fullPath, nil)
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
func (c *HTTPClient) UploadFile(uploadURL, method string, headers map[string]string, body io.Reader, contentLength int64) error {
	req, err := http.NewRequest(method, uploadURL, body)
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}

	req.ContentLength = contentLength
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetPackageStatus polls the status of a package.
func (c *HTTPClient) GetPackageStatus(appID, deploymentID, packageID string) (*PackageStatus, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s/status",
		appID, deploymentID, packageID)

	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result PackageStatus
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("getting package status: %w", err)
	}

	return &result, nil
}

// ListPackages returns all packages for a deployment.
func (c *HTTPClient) ListPackages(appID, deploymentID string) ([]Package, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages", appID, deploymentID)

	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result PackageListResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("listing packages: %w", err)
	}

	return result.Items, nil
}

// GetPackage returns a single package by ID.
func (c *HTTPClient) GetPackage(appID, deploymentID, packageID string) (*Package, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s",
		appID, deploymentID, packageID)

	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result Package
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("getting package: %w", err)
	}

	return &result, nil
}

// PatchPackage updates metadata on an existing package.
func (c *HTTPClient) PatchPackage(appID, deploymentID, packageID string, req PatchRequest) (*Package, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/packages/%s",
		appID, deploymentID, packageID)

	resp, err := c.doJSONRequest(http.MethodPatch, path, req)
	if err != nil {
		return nil, err
	}

	var result Package
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("patching package: %w", err)
	}

	return &result, nil
}

// Rollback sends a rollback request for a deployment.
func (c *HTTPClient) Rollback(appID, deploymentID string, req RollbackRequest) (*Package, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/rollback", appID, deploymentID)

	resp, err := c.doJSONRequest(http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	var result Package
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("rolling back deployment: %w", err)
	}

	return &result, nil
}

// Promote sends a promote request for a deployment.
func (c *HTTPClient) Promote(appID, deploymentID string, req PromoteRequest) (*Package, error) {
	path := fmt.Sprintf("/connected-apps/%s/code-push/deployments/%s/promote", appID, deploymentID)

	resp, err := c.doJSONRequest(http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}

	var result Package
	if err := decodeResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("promoting deployment: %w", err)
	}

	return &result, nil
}

func (c *HTTPClient) doJSONRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := c.BaseURL + path
	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request to %s: %w", path, err)
	}

	return resp, nil
}

func (c *HTTPClient) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	reqURL := c.BaseURL + path

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request to %s: %w", path, err)
	}

	return resp, nil
}

func decodeResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

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
