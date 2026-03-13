package aws

import (
	"context"
	"errors"
	"testing"
)

// mockCanaryClient implements CanaryClient for testing.
type mockCanaryClient struct {
	targetGroups    []TargetGroup
	describeTGErr   error
	createdTG       *TargetGroup
	createTGErr     error
	listeners       []Listener
	describeListErr error
	modifyErr       error
	modifyInput     *ModifyListenerInput // captures last ModifyListener call
	registerErr     error
}

func (m *mockCanaryClient) DescribeTargetGroups(_ context.Context, _ []string) ([]TargetGroup, error) {
	if m.describeTGErr != nil {
		return nil, m.describeTGErr
	}
	return m.targetGroups, nil
}

func (m *mockCanaryClient) CreateTargetGroup(_ context.Context, _ *CreateTargetGroupInput) (*TargetGroup, error) {
	if m.createTGErr != nil {
		return nil, m.createTGErr
	}
	return m.createdTG, nil
}

func (m *mockCanaryClient) DescribeListeners(_ context.Context, _ string) ([]Listener, error) {
	if m.describeListErr != nil {
		return nil, m.describeListErr
	}
	return m.listeners, nil
}

func (m *mockCanaryClient) ModifyListener(_ context.Context, input *ModifyListenerInput) error {
	if m.modifyErr != nil {
		return m.modifyErr
	}
	m.modifyInput = input
	return nil
}

func (m *mockCanaryClient) RegisterTargets(_ context.Context, _ string, _ []Target) error {
	return m.registerErr
}

func validCanaryConfig() CanaryConfig {
	return CanaryConfig{
		LoadBalancerARN:   "arn:aws:elasticloadbalancing:us-east-1:123456789:loadbalancer/app/my-alb/abc123",
		StableTargetGroup: "arn:aws:elasticloadbalancing:us-east-1:123456789:targetgroup/my-svc-stable/def456",
		ServiceName:       "my-svc",
		VPCID:             "vpc-abc123",
		Port:              8080,
		CanaryPercent:     10,
		HealthCheckPath:   "/health",
	}
}

func TestSetCanaryTraffic(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockCanaryClient{
			createdTG: &TargetGroup{
				ARN:  "arn:aws:elasticloadbalancing:us-east-1:123456789:targetgroup/my-svc-canary/ghi789",
				Name: "my-svc-canary",
			},
			listeners: []Listener{
				{ARN: "arn:aws:elasticloadbalancing:us-east-1:123456789:listener/app/my-alb/abc123/listener1", Port: 80},
			},
		}
		config := validCanaryConfig()

		result, err := SetCanaryTraffic(context.Background(), client, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.CanaryTargetGroupARN != client.createdTG.ARN {
			t.Errorf("CanaryTargetGroupARN = %q, want %q", result.CanaryTargetGroupARN, client.createdTG.ARN)
		}
		if result.StableTargetGroupARN != config.StableTargetGroup {
			t.Errorf("StableTargetGroupARN = %q, want %q", result.StableTargetGroupARN, config.StableTargetGroup)
		}
		if result.CanaryPercent != 10 {
			t.Errorf("CanaryPercent = %d, want 10", result.CanaryPercent)
		}
		if result.StablePercent != 90 {
			t.Errorf("StablePercent = %d, want 90", result.StablePercent)
		}

		if client.modifyInput == nil {
			t.Fatal("ModifyListener was not called")
		}
		if len(client.modifyInput.Actions) != 2 {
			t.Fatalf("ModifyListener actions = %d, want 2", len(client.modifyInput.Actions))
		}
		if client.modifyInput.Actions[0].Weight != 90 {
			t.Errorf("stable action weight = %d, want 90", client.modifyInput.Actions[0].Weight)
		}
		if client.modifyInput.Actions[1].Weight != 10 {
			t.Errorf("canary action weight = %d, want 10", client.modifyInput.Actions[1].Weight)
		}
	})

	t.Run("invalid percent zero", func(t *testing.T) {
		client := &mockCanaryClient{}
		config := validCanaryConfig()
		config.CanaryPercent = 0

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for canary percent 0")
		}
	})

	t.Run("invalid percent 100", func(t *testing.T) {
		client := &mockCanaryClient{}
		config := validCanaryConfig()
		config.CanaryPercent = 100

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for canary percent 100")
		}
	})

	t.Run("missing load balancer ARN", func(t *testing.T) {
		client := &mockCanaryClient{}
		config := validCanaryConfig()
		config.LoadBalancerARN = ""

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for empty load balancer ARN")
		}
	})

	t.Run("missing stable target group", func(t *testing.T) {
		client := &mockCanaryClient{}
		config := validCanaryConfig()
		config.StableTargetGroup = ""

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for empty stable target group")
		}
	})

	t.Run("missing service name", func(t *testing.T) {
		client := &mockCanaryClient{}
		config := validCanaryConfig()
		config.ServiceName = ""

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for empty service name")
		}
	})

	t.Run("missing VPC ID", func(t *testing.T) {
		client := &mockCanaryClient{}
		config := validCanaryConfig()
		config.VPCID = ""

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for empty VPC ID")
		}
	})

	t.Run("missing port", func(t *testing.T) {
		client := &mockCanaryClient{}
		config := validCanaryConfig()
		config.Port = 0

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for zero port")
		}
	})

	t.Run("create target group fails", func(t *testing.T) {
		client := &mockCanaryClient{
			createTGErr: errors.New("quota exceeded"),
		}
		config := validCanaryConfig()

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error when CreateTargetGroup fails")
		}
	})

	t.Run("no listeners found", func(t *testing.T) {
		client := &mockCanaryClient{
			createdTG: &TargetGroup{ARN: "arn:tg-canary", Name: "my-svc-canary"},
			listeners: []Listener{},
		}
		config := validCanaryConfig()

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error when no listeners found")
		}
	})

	t.Run("modify listener fails", func(t *testing.T) {
		client := &mockCanaryClient{
			createdTG: &TargetGroup{ARN: "arn:tg-canary", Name: "my-svc-canary"},
			listeners: []Listener{{ARN: "arn:listener1", Port: 80}},
			modifyErr: errors.New("permission denied"),
		}
		config := validCanaryConfig()

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error when ModifyListener fails")
		}
	})
}

func TestPromoteCanary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockCanaryClient{
			listeners: []Listener{
				{ARN: "arn:listener1", Port: 80},
			},
		}

		err := PromoteCanary(context.Background(), client,
			"arn:aws:elasticloadbalancing:us-east-1:123456789:loadbalancer/app/my-alb/abc123",
			"arn:aws:elasticloadbalancing:us-east-1:123456789:targetgroup/my-svc-canary/ghi789",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client.modifyInput == nil {
			t.Fatal("ModifyListener was not called")
		}
		if len(client.modifyInput.Actions) != 1 {
			t.Fatalf("ModifyListener actions = %d, want 1", len(client.modifyInput.Actions))
		}
		if client.modifyInput.Actions[0].Weight != 100 {
			t.Errorf("promoted weight = %d, want 100", client.modifyInput.Actions[0].Weight)
		}
		if client.modifyInput.Actions[0].TargetGroupARN != "arn:aws:elasticloadbalancing:us-east-1:123456789:targetgroup/my-svc-canary/ghi789" {
			t.Errorf("promoted target group ARN mismatch")
		}
	})

	t.Run("no listener found", func(t *testing.T) {
		client := &mockCanaryClient{
			listeners: []Listener{},
		}

		err := PromoteCanary(context.Background(), client, "arn:alb", "arn:tg-canary")
		if err == nil {
			t.Fatal("expected error when no listeners found")
		}
	})

	t.Run("empty load balancer ARN", func(t *testing.T) {
		client := &mockCanaryClient{}

		err := PromoteCanary(context.Background(), client, "", "arn:tg-canary")
		if err == nil {
			t.Fatal("expected error for empty load balancer ARN")
		}
	})

	t.Run("empty canary target group ARN", func(t *testing.T) {
		client := &mockCanaryClient{}

		err := PromoteCanary(context.Background(), client, "arn:alb", "")
		if err == nil {
			t.Fatal("expected error for empty canary target group ARN")
		}
	})

	t.Run("describe listeners fails", func(t *testing.T) {
		client := &mockCanaryClient{
			describeListErr: errors.New("network error"),
		}

		err := PromoteCanary(context.Background(), client, "arn:alb", "arn:tg-canary")
		if err == nil {
			t.Fatal("expected error when DescribeListeners fails")
		}
	})

	t.Run("modify listener fails", func(t *testing.T) {
		client := &mockCanaryClient{
			listeners: []Listener{{ARN: "arn:listener1", Port: 80}},
			modifyErr: errors.New("permission denied"),
		}

		err := PromoteCanary(context.Background(), client, "arn:alb", "arn:tg-canary")
		if err == nil {
			t.Fatal("expected error when ModifyListener fails")
		}
	})
}
