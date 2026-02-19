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

// Package represents a CodePush release in a deployment.
type Package struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	AppVersion    string `json:"app_version"`
	Description   string `json:"description"`
	Mandatory     bool   `json:"mandatory"`
	Disabled      bool   `json:"disabled"`
	Rollout       int    `json:"rollout"`
	DeploymentID  string `json:"deployment_id"`
	FileSizeBytes int64  `json:"file_size_bytes"`
}

// PackageListResponse wraps the list packages API response.
type PackageListResponse struct {
	Items []Package `json:"items"`
}

// RollbackOptions holds user-provided parameters for a rollback operation.
type RollbackOptions struct {
	AppID        string
	DeploymentID string
	Token        string
	TargetLabel  string // optional: specific label like "v3" to rollback to
}

// RollbackRequest is the JSON body sent to the rollback API endpoint.
type RollbackRequest struct {
	PackageID string `json:"package_id,omitempty"`
}

// RollbackResult is the output of a successful rollback.
type RollbackResult struct {
	PackageID    string `json:"package_id"`
	AppID        string `json:"app_id"`
	DeploymentID string `json:"deployment_id"`
	Label        string `json:"label"`
	AppVersion   string `json:"app_version"`
}

// PromoteOptions holds user-provided parameters for a promote operation.
type PromoteOptions struct {
	AppID              string
	SourceDeploymentID string
	DestDeploymentID   string
	Token              string
	Label              string // optional: specific label to promote from source
	AppVersion         string // optional: override target app version
	Description        string // optional: override description
	Mandatory          string // optional: "true"/"false" override
	Disabled           string // optional: "true"/"false" override
	Rollout            string // optional: "1"-"100" override
}

// PromoteRequest is the JSON body sent to the promote API endpoint.
type PromoteRequest struct {
	TargetDeploymentID string `json:"target_deployment_id"`
	PackageID          string `json:"package_id,omitempty"`
	AppVersion         string `json:"app_version,omitempty"`
	Description        string `json:"description,omitempty"`
	Disabled           string `json:"disabled,omitempty"`
	Mandatory          string `json:"mandatory,omitempty"`
	Rollout            string `json:"rollout,omitempty"`
}

// PromoteResult is the output of a successful promote.
type PromoteResult struct {
	PackageID        string `json:"package_id"`
	AppID            string `json:"app_id"`
	SourceDeployment string `json:"source_deployment_id"`
	DestDeployment   string `json:"dest_deployment_id"`
	Label            string `json:"label"`
	AppVersion       string `json:"app_version"`
	Description      string `json:"description"`
}

// Client defines the CodePush API operations.
type Client interface {
	ListDeployments(appID string) ([]Deployment, error)
	GetUploadURL(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error)
	UploadFile(uploadURL, method string, headers map[string]string, body io.Reader, contentLength int64) error
	GetPackageStatus(appID, deploymentID, packageID string) (*PackageStatus, error)
	ListPackages(appID, deploymentID string) ([]Package, error)
	Rollback(appID, deploymentID string, req RollbackRequest) (*Package, error)
	Promote(appID, deploymentID string, req PromoteRequest) (*Package, error)
}
