package codepush

import (
	"context"
	"io"
	"time"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

type mockClient struct {
	listDeploymentsFunc    func(appID string) ([]Deployment, error)
	createDeploymentFunc   func(appID string, req CreateDeploymentRequest) (*Deployment, error)
	getDeploymentFunc      func(appID, deploymentID string) (*Deployment, error)
	renameDeploymentFunc   func(appID, deploymentID string, req RenameDeploymentRequest) (*Deployment, error)
	deleteDeploymentFunc   func(appID, deploymentID string) error
	getUploadURLFunc       func(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error)
	uploadFileFunc         func(req UploadFileRequest) error
	getPackageStatusFunc   func(appID, deploymentID, packageID string) (*PackageStatus, error)
	listPackagesFunc       func(appID, deploymentID string) ([]Package, error)
	getPackageFunc         func(appID, deploymentID, packageID string) (*Package, error)
	patchPackageFunc       func(appID, deploymentID, packageID string, req PatchRequest) (*Package, error)
	deletePackageFunc      func(appID, deploymentID, packageID string) error
	rollbackFunc           func(appID, deploymentID string, req RollbackRequest) (*Package, error)
	promoteFunc            func(appID, deploymentID string, req PromoteRequest) (*Package, error)
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

func (m *mockClient) GetUploadURL(_ context.Context, appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error) {
	if m.getUploadURLFunc != nil {
		return m.getUploadURLFunc(appID, deploymentID, packageID, req)
	}
	return &UploadURLResponse{URL: "https://example.com/upload", Method: "PUT"}, nil
}

func (m *mockClient) UploadFile(_ context.Context, req UploadFileRequest) error {
	if m.uploadFileFunc != nil {
		return m.uploadFileFunc(req)
	}
	return nil
}

func (m *mockClient) GetPackageStatus(_ context.Context, appID, deploymentID, packageID string) (*PackageStatus, error) {
	if m.getPackageStatusFunc != nil {
		return m.getPackageStatusFunc(appID, deploymentID, packageID)
	}
	return &PackageStatus{PackageID: packageID, Status: StatusProcessedValid}, nil
}

func (m *mockClient) ListPackages(_ context.Context, appID, deploymentID string) ([]Package, error) {
	if m.listPackagesFunc != nil {
		return m.listPackagesFunc(appID, deploymentID)
	}
	return nil, nil
}

func (m *mockClient) GetPackage(_ context.Context, appID, deploymentID, packageID string) (*Package, error) {
	if m.getPackageFunc != nil {
		return m.getPackageFunc(appID, deploymentID, packageID)
	}
	return &Package{ID: packageID, Label: "v1"}, nil
}

func (m *mockClient) PatchPackage(_ context.Context, appID, deploymentID, packageID string, req PatchRequest) (*Package, error) {
	if m.patchPackageFunc != nil {
		return m.patchPackageFunc(appID, deploymentID, packageID, req)
	}
	return &Package{ID: packageID, Label: "v1"}, nil
}

func (m *mockClient) DeletePackage(_ context.Context, appID, deploymentID, packageID string) error {
	if m.deletePackageFunc != nil {
		return m.deletePackageFunc(appID, deploymentID, packageID)
	}
	return nil
}

func (m *mockClient) Rollback(_ context.Context, appID, deploymentID string, req RollbackRequest) (*Package, error) {
	if m.rollbackFunc != nil {
		return m.rollbackFunc(appID, deploymentID, req)
	}
	return &Package{ID: "pkg-new", Label: "v2"}, nil
}

func (m *mockClient) Promote(_ context.Context, appID, deploymentID string, req PromoteRequest) (*Package, error) {
	if m.promoteFunc != nil {
		return m.promoteFunc(appID, deploymentID, req)
	}
	return &Package{ID: "pkg-new", Label: "v1"}, nil
}

var testOut = output.NewTest(io.Discard)

var fastPollConfig = PollConfig{
	MaxAttempts: 3,
	Interval:    1 * time.Millisecond,
}
