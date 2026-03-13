package aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
)

type mockAutoscaleAPI struct {
	registerErr    error
	putPolicyErr   error
	registerInput  *applicationautoscaling.RegisterScalableTargetInput
	putPolicyInput *applicationautoscaling.PutScalingPolicyInput
}

func (m *mockAutoscaleAPI) RegisterScalableTarget(_ context.Context, input *applicationautoscaling.RegisterScalableTargetInput, _ ...func(*applicationautoscaling.Options)) (*applicationautoscaling.RegisterScalableTargetOutput, error) {
	m.registerInput = input
	if m.registerErr != nil {
		return nil, m.registerErr
	}
	return &applicationautoscaling.RegisterScalableTargetOutput{}, nil
}

func (m *mockAutoscaleAPI) PutScalingPolicy(_ context.Context, input *applicationautoscaling.PutScalingPolicyInput, _ ...func(*applicationautoscaling.Options)) (*applicationautoscaling.PutScalingPolicyOutput, error) {
	m.putPolicyInput = input
	if m.putPolicyErr != nil {
		return nil, m.putPolicyErr
	}
	return &applicationautoscaling.PutScalingPolicyOutput{}, nil
}

func TestConfigureAutoScaling(t *testing.T) {
	tests := []struct {
		name           string
		config         AutoScaleConfig
		registerErr    error
		putPolicyErr   error
		wantErr        string
		wantCPU        float64
		wantMin        int32
		wantMax        int32
		wantResourceID string
	}{
		{
			name: "success with default CPU",
			config: AutoScaleConfig{
				ServiceName:  "my-svc",
				ClusterName:  "my-cluster",
				MinInstances: 1,
				MaxInstances: 5,
			},
			wantCPU:        70,
			wantMin:        1,
			wantMax:        5,
			wantResourceID: "service/my-cluster/my-svc",
		},
		{
			name: "success with custom CPU target",
			config: AutoScaleConfig{
				ServiceName:      "my-svc",
				ClusterName:      "my-cluster",
				MinInstances:     2,
				MaxInstances:     10,
				TargetCPUPercent: 50,
			},
			wantCPU:        50,
			wantMin:        2,
			wantMax:        10,
			wantResourceID: "service/my-cluster/my-svc",
		},
		{
			name: "register scalable target error",
			config: AutoScaleConfig{
				ServiceName:  "my-svc",
				ClusterName:  "my-cluster",
				MinInstances: 1,
				MaxInstances: 5,
			},
			registerErr: errors.New("access denied"),
			wantErr:     "register scalable target",
		},
		{
			name: "put scaling policy error",
			config: AutoScaleConfig{
				ServiceName:  "my-svc",
				ClusterName:  "my-cluster",
				MinInstances: 1,
				MaxInstances: 5,
			},
			putPolicyErr: errors.New("limit exceeded"),
			wantErr:      "put scaling policy",
		},
		{
			name: "empty service name",
			config: AutoScaleConfig{
				ClusterName:  "my-cluster",
				MinInstances: 1,
				MaxInstances: 5,
			},
			wantErr: "service name must not be empty",
		},
		{
			name: "empty cluster name",
			config: AutoScaleConfig{
				ServiceName:  "my-svc",
				MinInstances: 1,
				MaxInstances: 5,
			},
			wantErr: "cluster name must not be empty",
		},
		{
			name: "min instances zero",
			config: AutoScaleConfig{
				ServiceName:  "my-svc",
				ClusterName:  "my-cluster",
				MinInstances: 0,
				MaxInstances: 5,
			},
			wantErr: "min instances must be at least 1",
		},
		{
			name: "max less than min",
			config: AutoScaleConfig{
				ServiceName:  "my-svc",
				ClusterName:  "my-cluster",
				MinInstances: 5,
				MaxInstances: 2,
			},
			wantErr: "max instances (2) must be >= min instances (5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAutoscaleAPI{
				registerErr:  tt.registerErr,
				putPolicyErr: tt.putPolicyErr,
			}
			adapter := &AutoScaleAdapter{client: mock}

			err := adapter.ConfigureAutoScaling(context.Background(), tt.config)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify RegisterScalableTarget input.
			if mock.registerInput == nil {
				t.Fatal("RegisterScalableTarget was not called")
			}
			if got := *mock.registerInput.MinCapacity; got != tt.wantMin {
				t.Errorf("MinCapacity = %d, want %d", got, tt.wantMin)
			}
			if got := *mock.registerInput.MaxCapacity; got != tt.wantMax {
				t.Errorf("MaxCapacity = %d, want %d", got, tt.wantMax)
			}
			if got := *mock.registerInput.ResourceId; got != tt.wantResourceID {
				t.Errorf("ResourceId = %q, want %q", got, tt.wantResourceID)
			}

			// Verify PutScalingPolicy input.
			if mock.putPolicyInput == nil {
				t.Fatal("PutScalingPolicy was not called")
			}
			if got := *mock.putPolicyInput.TargetTrackingScalingPolicyConfiguration.TargetValue; got != tt.wantCPU {
				t.Errorf("TargetValue = %f, want %f", got, tt.wantCPU)
			}
			wantPolicyName := tt.config.ServiceName + "-cpu-target-tracking"
			if got := *mock.putPolicyInput.PolicyName; got != wantPolicyName {
				t.Errorf("PolicyName = %q, want %q", got, wantPolicyName)
			}
		})
	}
}
