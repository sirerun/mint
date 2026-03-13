package azure

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers"
)

type mockAutoscaleAPI struct {
	createOrUpdateFunc func(ctx context.Context, rg, name string, envelope armappcontainers.ContainerApp, opts *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error)
}

func (m *mockAutoscaleAPI) CreateOrUpdate(ctx context.Context, rg, name string, envelope armappcontainers.ContainerApp, opts *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
	return m.createOrUpdateFunc(ctx, rg, name, envelope, opts)
}

func TestKEDAAutoScaler_InterfaceCompliance(t *testing.T) {
	var _ AutoScaler = (*KEDAAutoScaler)(nil)
}

func TestKEDAAutoScaler_ConfigureAutoScaling(t *testing.T) {
	tests := []struct {
		name    string
		config  AutoScaleConfig
		apiErr  error
		wantErr string
		check   func(t *testing.T, rg, appName string, envelope armappcontainers.ContainerApp)
	}{
		{
			name: "success with default concurrent requests",
			config: AutoScaleConfig{
				AppName:       "my-app",
				ResourceGroup: "my-rg",
				MinReplicas:   1,
				MaxReplicas:   10,
			},
			check: func(t *testing.T, rg, appName string, envelope armappcontainers.ContainerApp) {
				t.Helper()
				if rg != "my-rg" {
					t.Errorf("resourceGroup = %q, want %q", rg, "my-rg")
				}
				if appName != "my-app" {
					t.Errorf("appName = %q, want %q", appName, "my-app")
				}
				scale := envelope.Properties.Template.Scale
				if *scale.MinReplicas != 1 {
					t.Errorf("MinReplicas = %d, want 1", *scale.MinReplicas)
				}
				if *scale.MaxReplicas != 10 {
					t.Errorf("MaxReplicas = %d, want 10", *scale.MaxReplicas)
				}
				if len(scale.Rules) != 1 {
					t.Fatalf("expected 1 scale rule, got %d", len(scale.Rules))
				}
				rule := scale.Rules[0]
				if *rule.Name != "http-concurrency" {
					t.Errorf("rule name = %q, want %q", *rule.Name, "http-concurrency")
				}
				if rule.HTTP == nil {
					t.Fatal("expected HTTP scale rule")
				}
				val := rule.HTTP.Metadata["concurrentRequests"]
				if val == nil || *val != "100" {
					t.Errorf("concurrentRequests = %v, want %q", val, "100")
				}
			},
		},
		{
			name: "success with custom concurrent requests",
			config: AutoScaleConfig{
				AppName:            "my-app",
				ResourceGroup:      "my-rg",
				MinReplicas:        0,
				MaxReplicas:        5,
				ConcurrentRequests: 50,
			},
			check: func(t *testing.T, _, _ string, envelope armappcontainers.ContainerApp) {
				t.Helper()
				scale := envelope.Properties.Template.Scale
				if *scale.MinReplicas != 0 {
					t.Errorf("MinReplicas = %d, want 0", *scale.MinReplicas)
				}
				if *scale.MaxReplicas != 5 {
					t.Errorf("MaxReplicas = %d, want 5", *scale.MaxReplicas)
				}
				val := scale.Rules[0].HTTP.Metadata["concurrentRequests"]
				if val == nil || *val != "50" {
					t.Errorf("concurrentRequests = %v, want %q", val, "50")
				}
			},
		},
		{
			name: "API update error",
			config: AutoScaleConfig{
				AppName:       "my-app",
				ResourceGroup: "my-rg",
				MinReplicas:   1,
				MaxReplicas:   3,
			},
			apiErr:  errors.New("conflict: resource busy"),
			wantErr: "configure auto-scaling for my-app",
		},
		{
			name: "empty app name",
			config: AutoScaleConfig{
				ResourceGroup: "my-rg",
				MinReplicas:   1,
				MaxReplicas:   3,
			},
			wantErr: "app name must not be empty",
		},
		{
			name: "empty resource group",
			config: AutoScaleConfig{
				AppName:     "my-app",
				MinReplicas: 1,
				MaxReplicas: 3,
			},
			wantErr: "resource group must not be empty",
		},
		{
			name: "negative min replicas",
			config: AutoScaleConfig{
				AppName:       "my-app",
				ResourceGroup: "my-rg",
				MinReplicas:   -1,
				MaxReplicas:   3,
			},
			wantErr: "min replicas must be non-negative",
		},
		{
			name: "zero max replicas",
			config: AutoScaleConfig{
				AppName:       "my-app",
				ResourceGroup: "my-rg",
				MinReplicas:   0,
				MaxReplicas:   0,
			},
			wantErr: "max replicas must be at least 1",
		},
		{
			name: "min exceeds max",
			config: AutoScaleConfig{
				AppName:       "my-app",
				ResourceGroup: "my-rg",
				MinReplicas:   5,
				MaxReplicas:   3,
			},
			wantErr: "min replicas (5) must not exceed max replicas (3)",
		},
		{
			name: "negative concurrent requests",
			config: AutoScaleConfig{
				AppName:            "my-app",
				ResourceGroup:      "my-rg",
				MinReplicas:        1,
				MaxReplicas:        3,
				ConcurrentRequests: -1,
			},
			wantErr: "concurrent requests must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRG, capturedApp string
			var capturedEnvelope armappcontainers.ContainerApp

			api := &mockAutoscaleAPI{
				createOrUpdateFunc: func(_ context.Context, rg, name string, envelope armappcontainers.ContainerApp, _ *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
					capturedRG = rg
					capturedApp = name
					capturedEnvelope = envelope
					if tt.apiErr != nil {
						return nil, tt.apiErr
					}
					return &armappcontainers.ContainerAppsClientCreateOrUpdateResponse{}, nil
				},
			}

			scaler := &KEDAAutoScaler{API: api}
			err := scaler.ConfigureAutoScaling(context.Background(), tt.config)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.check != nil {
				tt.check(t, capturedRG, capturedApp, capturedEnvelope)
			}
		})
	}
}

func TestDefaultConcurrentRequests(t *testing.T) {
	if DefaultConcurrentRequests != 100 {
		t.Fatalf("DefaultConcurrentRequests = %d, want 100", DefaultConcurrentRequests)
	}
}
