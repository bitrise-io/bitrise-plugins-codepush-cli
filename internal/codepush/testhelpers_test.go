package codepush

import (
	"context"
	"io"
	"time"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

type mockClient struct {
	listDeploymentsFunc  func(appID string) ([]Deployment, error)
	createDeploymentFunc func(appID string, req CreateDeploymentRequest) (*Deployment, error)
	getDeploymentFunc    func(appID, deploymentID string) (*Deployment, error)
	renameDeploymentFunc func(appID, deploymentID string, req RenameDeploymentRequest) (*Deployment, error)
	deleteDeploymentFunc func(appID, deploymentID string) error
	getUploadURLFunc     func(appID, deploymentID, updateID string, req UploadURLRequest) (*UploadURLResponse, error)
	uploadFileFunc       func(req UploadFileRequest) error
	getUpdateStatusFunc  func(appID, deploymentID, updateID string) (*UpdateStatus, error)
	listUpdatesFunc      func(appID, deploymentID string) ([]Update, error)
	getUpdateFunc        func(appID, deploymentID, updateID string) (*Update, error)
	patchUpdateFunc      func(appID, deploymentID, updateID string, req PatchRequest) (*Update, error)
	deleteUpdateFunc     func(appID, deploymentID, updateID string) error
	rollbackFunc         func(appID, deploymentID string, req RollbackRequest) (*Update, error)
	promoteFunc          func(appID, deploymentID string, req PromoteRequest) (*Update, error)
}

func (m *mockClient) ListDeployments(_ context.Context, appID string) ([]Deployment, error) {
	if m.listDeploymentsFunc != nil {
		return m.listDeploymentsFunc(appID)
	}
	return nil, nil
}

func (m *mockClient) CreateDeployment(_ context.Context, appID string, req CreateDeploymentRequest) (*Deployment, error) {
	if m.createDeploymentFunc != nil {
		return m.createDeploymentFunc(appID, req)
	}
	return &Deployment{ID: "dep-new", Name: req.Name}, nil
}

func (m *mockClient) GetDeployment(_ context.Context, appID, deploymentID string) (*Deployment, error) {
	if m.getDeploymentFunc != nil {
		return m.getDeploymentFunc(appID, deploymentID)
	}
	return &Deployment{ID: deploymentID, Name: "Test"}, nil
}

func (m *mockClient) RenameDeployment(_ context.Context, appID, deploymentID string, req RenameDeploymentRequest) (*Deployment, error) {
	if m.renameDeploymentFunc != nil {
		return m.renameDeploymentFunc(appID, deploymentID, req)
	}
	return &Deployment{ID: deploymentID, Name: req.Name}, nil
}

func (m *mockClient) DeleteDeployment(_ context.Context, appID, deploymentID string) error {
	if m.deleteDeploymentFunc != nil {
		return m.deleteDeploymentFunc(appID, deploymentID)
	}
	return nil
}

func (m *mockClient) GetUploadURL(_ context.Context, appID, deploymentID, updateID string, req UploadURLRequest) (*UploadURLResponse, error) {
	if m.getUploadURLFunc != nil {
		return m.getUploadURLFunc(appID, deploymentID, updateID, req)
	}
	return &UploadURLResponse{URL: "https://example.com/upload", Method: "PUT"}, nil
}

func (m *mockClient) UploadFile(_ context.Context, req UploadFileRequest) error {
	if m.uploadFileFunc != nil {
		return m.uploadFileFunc(req)
	}
	return nil
}

func (m *mockClient) GetUpdateStatus(_ context.Context, appID, deploymentID, updateID string) (*UpdateStatus, error) {
	if m.getUpdateStatusFunc != nil {
		return m.getUpdateStatusFunc(appID, deploymentID, updateID)
	}
	return &UpdateStatus{UpdateID: updateID, Status: StatusProcessedValid}, nil
}

func (m *mockClient) ListUpdates(_ context.Context, appID, deploymentID string) ([]Update, error) {
	if m.listUpdatesFunc != nil {
		return m.listUpdatesFunc(appID, deploymentID)
	}
	return nil, nil
}

func (m *mockClient) GetUpdate(_ context.Context, appID, deploymentID, updateID string) (*Update, error) {
	if m.getUpdateFunc != nil {
		return m.getUpdateFunc(appID, deploymentID, updateID)
	}
	return &Update{ID: updateID, Label: "v1"}, nil
}

func (m *mockClient) PatchUpdate(_ context.Context, appID, deploymentID, updateID string, req PatchRequest) (*Update, error) {
	if m.patchUpdateFunc != nil {
		return m.patchUpdateFunc(appID, deploymentID, updateID, req)
	}
	return &Update{ID: updateID, Label: "v1"}, nil
}

func (m *mockClient) DeleteUpdate(_ context.Context, appID, deploymentID, updateID string) error {
	if m.deleteUpdateFunc != nil {
		return m.deleteUpdateFunc(appID, deploymentID, updateID)
	}
	return nil
}

func (m *mockClient) Rollback(_ context.Context, appID, deploymentID string, req RollbackRequest) (*Update, error) {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(appID, deploymentID, req)
	}
	return &Update{ID: "pkg-new", Label: "v2"}, nil
}

func (m *mockClient) Promote(_ context.Context, appID, deploymentID string, req PromoteRequest) (*Update, error) {
	if m.promoteFunc != nil {
		return m.promoteFunc(appID, deploymentID, req)
	}
	return &Update{ID: "pkg-new", Label: "v1"}, nil
}

var testOut = output.NewTest(io.Discard)

var fastPollConfig = PollConfig{
	MaxAttempts: 3,
	Interval:    1 * time.Millisecond,
}
