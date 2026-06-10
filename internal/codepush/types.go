package codepush

import (
	"context"
	"encoding/json"
	"fmt"
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
	Rollout      float64
	BundlePath   string
}

// UploadURLRequest represents the query parameters for requesting an upload URL.
// Rollout uses a pointer to distinguish "not set" (nil, param omitted) from zero (0% rollout sent explicitly).
type UploadURLRequest struct {
	AppVersion    string
	FileName      string
	FileSizeBytes int64
	Description   string
	Mandatory     bool
	Disabled      bool
	Rollout       *float64
}

// HeaderMap is a map[string]string that can unmarshal from either a JSON object
// or a JSON array of {"key": "...", "value": "..."} objects, as returned by
// the upload-url API endpoint.
type HeaderMap map[string]string

// UnmarshalJSON handles both object and array-of-objects formats.
func (h *HeaderMap) UnmarshalJSON(data []byte) error {
	// Try object format first: {"Content-Type": "application/zip"}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err == nil {
		*h = m
		return nil
	}

	// Try array-of-objects format: [{"key": "k", "value": "v"}] or [{"name": "k", "value": "v"}]
	var arr []struct {
		Key   string `json:"key"`
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("headers: expected object or array of {key, value}, got %s", string(data))
	}

	result := make(map[string]string, len(arr))
	for _, item := range arr {
		k := item.Key
		if k == "" {
			k = item.Name
		}
		if k == "" {
			continue
		}
		result[k] = item.Value
	}
	*h = result
	return nil
}

// UploadURLResponse is returned by the GET upload-url endpoint.
type UploadURLResponse struct {
	URL     string    `json:"url"`
	Method  string    `json:"method"`
	Headers HeaderMap `json:"headers"`
}

// UploadFileRequest holds all parameters needed to upload a file.
type UploadFileRequest struct {
	URL           string
	Method        string
	Headers       map[string]string
	Body          io.Reader
	ContentLength int64
}


// UpdateStatus is returned by the GET status endpoint.
type UpdateStatus struct {
	UpdateID     string `json:"update_id"`
	Status       string `json:"status"`
	StatusReason string `json:"status_reason"`
}

// Deployment represents a CodePush deployment.
type Deployment struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	CreatedAt       string  `json:"created_at,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
	Key             string  `json:"key,omitempty"`
	NumberOfUpdates int     `json:"number_of_updates,omitempty"`
	LatestUpdate    *Update `json:"update,omitempty"`
}

// CreateDeploymentRequest is the JSON body for creating a deployment.
type CreateDeploymentRequest struct {
	Name  string `json:"name"`
	Key   string `json:"key,omitempty"`
	AppID string `json:"app_id"`
}

// RenameDeploymentRequest is the JSON body for renaming a deployment.
type RenameDeploymentRequest struct {
	Name string `json:"name"`
}

// DeploymentListResponse wraps the list deployments API response.
type DeploymentListResponse struct {
	Items []Deployment `json:"items"`
}

// PushResult is the output of a successful push.
type PushResult struct {
	UpdateID      string  `json:"package_id"`
	AppID         string  `json:"app_id"`
	DeploymentID  string  `json:"deployment_id"`
	AppVersion    string  `json:"app_version"`
	Status        string  `json:"status"`
	FileSizeBytes int64   `json:"file_size_bytes"`
	Rollout       float64 `json:"rollout"`
}

// PollConfig controls the polling behavior when waiting for update processing.
type PollConfig struct {
	MaxAttempts int
	Interval    time.Duration
}

// DefaultPollConfig is used in production.
var DefaultPollConfig = PollConfig{
	MaxAttempts: 60,
	Interval:    2 * time.Second,
}

// Status constants for update processing.
const (
	StatusCreated        = "created"
	StatusUploaded       = "uploaded"
	StatusProcessedValid = "processed_valid"
	StatusProcessedError = "processed_invalid"
)

// UpdateCreator identifies the user who created an update.
type UpdateCreator struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

// Update represents a CodePush release in a deployment.
type Update struct {
	ID            string         `json:"id"`
	Label         string         `json:"label"`
	AppVersion    string         `json:"app_version"`
	Description   string         `json:"description"`
	Mandatory     bool           `json:"mandatory"`
	Disabled      bool           `json:"disabled"`
	Rollout       float64        `json:"rollout"`
	Signed        bool           `json:"signed,omitempty"`
	UpdateVersion string         `json:"update_version,omitempty"`
	DeploymentID  string         `json:"deployment_id,omitempty"`
	FileSizeBytes int64          `json:"file_size_bytes"`
	CreatedAt     string         `json:"created_at,omitempty"`
	UpdatedAt     string         `json:"updated_at,omitempty"`
	Hash          string         `json:"hash,omitempty"`
	FileName      string         `json:"file_name,omitempty"`
	CreatedBy     *UpdateCreator `json:"created_by,omitempty"`
	UpdatedBy     *UpdateCreator `json:"updated_by,omitempty"`
}

// UpdateListResponse wraps the list updates API response.
type UpdateListResponse struct {
	Items []Update `json:"items"`
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
	UpdateID     string `json:"package_id"`
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
	Rollout            string // optional: "0"-"100" override
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
	UpdateID         string `json:"package_id"`
	AppID            string `json:"app_id"`
	SourceDeployment string `json:"source_deployment_id"`
	DestDeployment   string `json:"dest_deployment_id"`
	Label            string `json:"label"`
	AppVersion       string `json:"app_version"`
	Description      string `json:"description"`
}

// PatchOptions holds user-provided parameters for a patch operation.
type PatchOptions struct {
	AppID        string
	DeploymentID string
	Token        string
	Label        string // optional: specific label like "v5", defaults to latest
	Rollout      string // optional: "0"-"100"
	Mandatory    string // optional: "true"/"false"
	Disabled     string // optional: "true"/"false"
}

// PatchRequest is the JSON body sent to the PATCH update API endpoint.
// Rollout uses a pointer to distinguish "not set" from zero; Mandatory and Disabled use omitempty strings.
type PatchRequest struct {
	Rollout   *float64 `json:"rollout,omitempty"`
	Mandatory string   `json:"mandatory,omitempty"`
	Disabled  string   `json:"disabled,omitempty"`
}

// PatchResult is the output of a successful patch.
type PatchResult struct {
	UpdateID     string  `json:"package_id"`
	AppID        string  `json:"app_id"`
	DeploymentID string  `json:"deployment_id"`
	Label        string  `json:"label"`
	AppVersion   string  `json:"app_version"`
	Mandatory    bool    `json:"mandatory"`
	Disabled     bool    `json:"disabled"`
	Rollout      float64 `json:"rollout"`
	Description  string  `json:"description"`
}

// Client defines the CodePush API operations.
type Client interface {
	ListDeployments(ctx context.Context, appID string) ([]Deployment, error)
	CreateDeployment(ctx context.Context, req CreateDeploymentRequest) (*Deployment, error)
	GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error)
	RenameDeployment(ctx context.Context, deploymentID string, req RenameDeploymentRequest) (*Deployment, error)
	DeleteDeployment(ctx context.Context, deploymentID string) error
	GetUploadURL(ctx context.Context, deploymentID, updateID string, req UploadURLRequest) (*UploadURLResponse, error)
	UploadFile(ctx context.Context, req UploadFileRequest) error
	GetUpdateStatus(ctx context.Context, updateID string) (*UpdateStatus, error)
	ListUpdates(ctx context.Context, deploymentID string) ([]Update, error)
	GetUpdate(ctx context.Context, updateID string) (*Update, error)
	PatchUpdate(ctx context.Context, updateID string, req PatchRequest) (*Update, error)
	DeleteUpdate(ctx context.Context, updateID string) error
	Rollback(ctx context.Context, deploymentID string, req RollbackRequest) (*Update, error)
	Promote(ctx context.Context, deploymentID string, req PromoteRequest) (*Update, error)
}
