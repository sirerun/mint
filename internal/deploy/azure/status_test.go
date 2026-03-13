package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type mockContainerAppClientForStatus struct {
	app         *ContainerApp
	appErr      error
	revisions   []Revision
	revisionErr error
}

func (m *mockContainerAppClientForStatus) CreateOrUpdateApp(_ context.Context, _ *CreateOrUpdateAppInput) (*ContainerApp, error) {
	return nil, nil
}

func (m *mockContainerAppClientForStatus) GetApp(_ context.Context, _, _ string) (*ContainerApp, error) {
	if m.appErr != nil {
		return nil, m.appErr
	}
	return m.app, nil
}

func (m *mockContainerAppClientForStatus) ListRevisions(_ context.Context, _, _ string) ([]Revision, error) {
	if m.revisionErr != nil {
		return nil, m.revisionErr
	}
	return m.revisions, nil
}

func (m *mockContainerAppClientForStatus) UpdateTrafficSplit(_ context.Context, _, _ string, _ []TrafficWeight) error {
	return nil
}

func TestGetContainerAppStatus_Success(t *testing.T) {
	client := &mockContainerAppClientForStatus{
		app: &ContainerApp{
			Name:              "my-app",
			FQDN:              "my-app.azurecontainerapps.io",
			ProvisioningState: "Succeeded",
			LatestRevision:    "my-app--rev3",
		},
		revisions: []Revision{
			{Name: "my-app--rev3", Active: true, TrafficWeight: 80, CreatedTime: "2026-03-13T10:00:00Z"},
			{Name: "my-app--rev2", Active: true, TrafficWeight: 20, CreatedTime: "2026-03-12T10:00:00Z"},
			{Name: "my-app--rev1", Active: false, TrafficWeight: 0, CreatedTime: "2026-03-11T10:00:00Z"},
		},
	}

	result, err := GetContainerAppStatus(context.Background(), client, "my-rg", "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ServiceName != "my-app" {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, "my-app")
	}
	if result.ResourceGroup != "my-rg" {
		t.Errorf("ResourceGroup = %q, want %q", result.ResourceGroup, "my-rg")
	}
	if result.FQDN != "my-app.azurecontainerapps.io" {
		t.Errorf("FQDN = %q, want %q", result.FQDN, "my-app.azurecontainerapps.io")
	}
	if result.ProvisioningState != "Succeeded" {
		t.Errorf("ProvisioningState = %q, want %q", result.ProvisioningState, "Succeeded")
	}
	if result.LatestRevision != "my-app--rev3" {
		t.Errorf("LatestRevision = %q, want %q", result.LatestRevision, "my-app--rev3")
	}
	if len(result.Revisions) != 3 {
		t.Fatalf("Revisions count = %d, want 3", len(result.Revisions))
	}
	if result.Revisions[0].Name != "my-app--rev3" {
		t.Errorf("Revisions[0].Name = %q, want %q", result.Revisions[0].Name, "my-app--rev3")
	}
	if len(result.TrafficWeights) != 2 {
		t.Fatalf("TrafficWeights count = %d, want 2", len(result.TrafficWeights))
	}
	if result.TrafficWeights[0].Weight != 80 {
		t.Errorf("TrafficWeights[0].Weight = %d, want 80", result.TrafficWeights[0].Weight)
	}
	if result.TrafficWeights[1].Weight != 20 {
		t.Errorf("TrafficWeights[1].Weight = %d, want 20", result.TrafficWeights[1].Weight)
	}
}

func TestGetContainerAppStatus_AppNotFound(t *testing.T) {
	client := &mockContainerAppClientForStatus{
		appErr: ErrAppNotFound,
	}

	_, err := GetContainerAppStatus(context.Background(), client, "my-rg", "missing-app")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting app") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "getting app")
	}
}

func TestGetContainerAppStatus_RevisionListFailure_PartialResult(t *testing.T) {
	client := &mockContainerAppClientForStatus{
		app: &ContainerApp{
			Name:              "my-app",
			FQDN:              "my-app.azurecontainerapps.io",
			ProvisioningState: "Succeeded",
			LatestRevision:    "my-app--rev1",
		},
		revisionErr: fmt.Errorf("access denied"),
	}

	result, err := GetContainerAppStatus(context.Background(), client, "my-rg", "my-app")
	if err == nil {
		t.Fatal("expected error for revision list failure")
	}
	if !strings.Contains(err.Error(), "listing revisions") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "listing revisions")
	}
	if result == nil {
		t.Fatal("expected partial result, got nil")
	}
	if result.ServiceName != "my-app" {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, "my-app")
	}
	if result.Revisions != nil {
		t.Errorf("Revisions = %v, want nil on revision list failure", result.Revisions)
	}
}

func TestFormatStatus_HumanReadable(t *testing.T) {
	result := &StatusResult{
		ServiceName:       "my-app",
		ResourceGroup:     "my-rg",
		FQDN:              "my-app.azurecontainerapps.io",
		ProvisioningState: "Succeeded",
		LatestRevision:    "my-app--rev2",
		Revisions: []RevisionInfo{
			{Name: "my-app--rev2", CreatedTime: "2026-03-13T10:00:00Z", ProvisioningState: "Provisioned", RunningState: "Running", Replicas: 3},
			{Name: "my-app--rev1", CreatedTime: "2026-03-12T10:00:00Z", ProvisioningState: "Provisioned", RunningState: "Running", Replicas: 1},
		},
		TrafficWeights: []TrafficWeight{
			{RevisionName: "my-app--rev2", Weight: 100},
		},
	}

	output := FormatStatus(result, false)

	checks := []string{
		"my-app",
		"my-rg",
		"my-app.azurecontainerapps.io",
		"Succeeded",
		"my-app--rev2",
		"my-app--rev1",
		"NAME",
		"CREATED",
		"REVISION",
		"WEIGHT",
		"100%",
	}

	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("text output missing %q\nOutput:\n%s", want, output)
		}
	}
}

func TestFormatStatus_HumanReadable_EmptyRevisions(t *testing.T) {
	result := &StatusResult{
		ServiceName:       "my-app",
		ResourceGroup:     "my-rg",
		FQDN:              "my-app.azurecontainerapps.io",
		ProvisioningState: "Succeeded",
		LatestRevision:    "my-app--rev1",
		Revisions:         []RevisionInfo{},
		TrafficWeights:    []TrafficWeight{},
	}

	output := FormatStatus(result, false)
	if strings.Contains(output, "Revisions:") {
		t.Error("output should not contain Revisions header when revisions are empty")
	}
	if strings.Contains(output, "Traffic:") {
		t.Error("output should not contain Traffic header when traffic weights are empty")
	}
	if !strings.Contains(output, "my-app") {
		t.Error("output missing service name")
	}
}

func TestFormatStatus_JSON(t *testing.T) {
	result := &StatusResult{
		ServiceName:       "test-app",
		ResourceGroup:     "test-rg",
		FQDN:              "test-app.azurecontainerapps.io",
		ProvisioningState: "Succeeded",
		LatestRevision:    "test-app--rev1",
		Revisions: []RevisionInfo{
			{Name: "test-app--rev1", CreatedTime: "2026-03-13T10:00:00Z"},
		},
		TrafficWeights: []TrafficWeight{
			{RevisionName: "test-app--rev1", Weight: 100},
		},
	}

	output := FormatStatus(result, true)

	var parsed StatusResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput:\n%s", err, output)
	}

	if parsed.ServiceName != "test-app" {
		t.Errorf("parsed ServiceName = %q, want %q", parsed.ServiceName, "test-app")
	}
	if parsed.ResourceGroup != "test-rg" {
		t.Errorf("parsed ResourceGroup = %q, want %q", parsed.ResourceGroup, "test-rg")
	}
	if len(parsed.Revisions) != 1 {
		t.Fatalf("parsed Revisions count = %d, want 1", len(parsed.Revisions))
	}
	if parsed.Revisions[0].Name != "test-app--rev1" {
		t.Errorf("parsed Revisions[0].Name = %q, want %q", parsed.Revisions[0].Name, "test-app--rev1")
	}
	if len(parsed.TrafficWeights) != 1 {
		t.Fatalf("parsed TrafficWeights count = %d, want 1", len(parsed.TrafficWeights))
	}
	if parsed.TrafficWeights[0].Weight != 100 {
		t.Errorf("parsed TrafficWeights[0].Weight = %d, want 100", parsed.TrafficWeights[0].Weight)
	}
}

func TestFormatStatus_JSONWithNilRevisions(t *testing.T) {
	result := &StatusResult{
		ServiceName: "svc",
		Revisions:   nil,
	}
	output := FormatStatus(result, true)
	var parsed StatusResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.ServiceName != "svc" {
		t.Errorf("ServiceName = %q, want %q", parsed.ServiceName, "svc")
	}
}

func TestFormatStatus_JSONMarshalError(t *testing.T) {
	original := jsonMarshalIndent
	t.Cleanup(func() { jsonMarshalIndent = original })

	jsonMarshalIndent = func(_ any, _ string, _ string) ([]byte, error) {
		return nil, fmt.Errorf("forced marshal error")
	}

	result := &StatusResult{ServiceName: "svc"}
	output := FormatStatus(result, true)
	if !strings.Contains(output, "error") {
		t.Errorf("expected error JSON fallback, got %q", output)
	}
	if !strings.Contains(output, "forced marshal error") {
		t.Errorf("expected error message in output, got %q", output)
	}
}
