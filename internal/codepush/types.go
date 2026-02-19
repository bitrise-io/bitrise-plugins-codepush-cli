package codepush

import (
	"io"
	"time"
)

// PushOptions holds user-provided parameters for a push operation.
type PushOptions struct {
	AppID        string
	DeploymentID string
	Token        string
	APIURL       string
	AppVersion   string
	Description  string
	Mandatory    bool
	Disabled     bool
	Rollout      int
	BundlePath   string
}

// UploadURLRequest represents the query parameters for requesting an upload URL.
type UploadURLRequest struct {
	AppVersion    string
	FileName      string
	FileSizeBytes int64
	Description   string
	Mandatory     bool
	Disabled      bool
	Rollout       int
}

// UploadURLResponse is returned by the GET upload-url endpoint.
type UploadURLResponse struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

// PackageStatus is returned by the GET status endpoint.
type PackageStatus struct {
	PackageID    string `json:"package_id"`
	Status       string `json:"status"`
	StatusReason string `json:"status_reason"`
}

// Deployment represents a CodePush deployment.
type Deployment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DeploymentListResponse wraps the list deployments API response.
type DeploymentListResponse struct {
	Items []Deployment `json:"items"`
}

// PushResult is the output of a successful push.
type PushResult struct {
	PackageID     string `json:"package_id"`
	AppID         string `json:"app_id"`
	DeploymentID  string `json:"deployment_id"`
	AppVersion    string `json:"app_version"`
	Status        string `json:"status"`
	FileSizeBytes int64  `json:"file_size_bytes"`
}

// PollConfig controls the polling behavior when waiting for package processing.
type PollConfig struct {
	MaxAttempts int
	Interval    time.Duration
}

// DefaultPollConfig is used in production.
var DefaultPollConfig = PollConfig{
	MaxAttempts: 60,
	Interval:    2 * time.Second,
}

// Status constants for package processing.
const (
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

// Client defines the CodePush API operations needed for push.
type Client interface {
	ListDeployments(appID string) ([]Deployment, error)
	GetUploadURL(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error)
	UploadFile(uploadURL, method string, headers map[string]string, body io.Reader, contentLength int64) error
	GetPackageStatus(appID, deploymentID, packageID string) (*PackageStatus, error)
}
