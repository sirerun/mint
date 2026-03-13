package aws

import (
	"context"
	"errors"
	"testing"
)

// mockALBClient is a test double for ALBClient.
type mockALBClient struct {
	createLoadBalancer    func(ctx context.Context, input *CreateLoadBalancerInput) (*LoadBalancer, error)
	describeLoadBalancers func(ctx context.Context, names []string) ([]LoadBalancer, error)
	createTargetGroup     func(ctx context.Context, input *CreateTargetGroupInput) (*TargetGroup, error)
	describeTargetGroups  func(ctx context.Context, names []string) ([]TargetGroup, error)
	createListener        func(ctx context.Context, input *CreateListenerInput) (*Listener, error)
	describeListeners     func(ctx context.Context, loadBalancerARN string) ([]Listener, error)
	modifyListener        func(ctx context.Context, input *ModifyListenerInput) error
	registerTargets       func(ctx context.Context, targetGroupARN string, targets []Target) error
	describeTargetHealth  func(ctx context.Context, targetGroupARN string) ([]TargetHealth, error)
}

func (m *mockALBClient) CreateLoadBalancer(ctx context.Context, input *CreateLoadBalancerInput) (*LoadBalancer, error) {
	return m.createLoadBalancer(ctx, input)
}

func (m *mockALBClient) DescribeLoadBalancers(ctx context.Context, names []string) ([]LoadBalancer, error) {
	return m.describeLoadBalancers(ctx, names)
}

func (m *mockALBClient) CreateTargetGroup(ctx context.Context, input *CreateTargetGroupInput) (*TargetGroup, error) {
	return m.createTargetGroup(ctx, input)
}

func (m *mockALBClient) DescribeTargetGroups(ctx context.Context, names []string) ([]TargetGroup, error) {
	return m.describeTargetGroups(ctx, names)
}

func (m *mockALBClient) CreateListener(ctx context.Context, input *CreateListenerInput) (*Listener, error) {
	return m.createListener(ctx, input)
}

func (m *mockALBClient) DescribeListeners(ctx context.Context, loadBalancerARN string) ([]Listener, error) {
	return m.describeListeners(ctx, loadBalancerARN)
}

func (m *mockALBClient) ModifyListener(ctx context.Context, input *ModifyListenerInput) error {
	return m.modifyListener(ctx, input)
}

func (m *mockALBClient) RegisterTargets(ctx context.Context, targetGroupARN string, targets []Target) error {
	return m.registerTargets(ctx, targetGroupARN, targets)
}

func (m *mockALBClient) DescribeTargetHealth(ctx context.Context, targetGroupARN string) ([]TargetHealth, error) {
	return m.describeTargetHealth(ctx, targetGroupARN)
}

func TestEnsureLoadBalancer_AllNew(t *testing.T) {
	mock := &mockALBClient{
		describeLoadBalancers: func(_ context.Context, _ []string) ([]LoadBalancer, error) {
			return nil, errors.New("not found")
		},
		createLoadBalancer: func(_ context.Context, input *CreateLoadBalancerInput) (*LoadBalancer, error) {
			return &LoadBalancer{ARN: "arn:lb:1", DNSName: "lb.example.com", Name: input.Name}, nil
		},
		describeTargetGroups: func(_ context.Context, _ []string) ([]TargetGroup, error) {
			return nil, errors.New("not found")
		},
		createTargetGroup: func(_ context.Context, input *CreateTargetGroupInput) (*TargetGroup, error) {
			if input.HealthCheckPath != "/health" {
				t.Errorf("expected health check path /health, got %s", input.HealthCheckPath)
			}
			return &TargetGroup{ARN: "arn:tg:1", Name: input.Name}, nil
		},
		describeListeners: func(_ context.Context, _ string) ([]Listener, error) {
			return nil, errors.New("not found")
		},
		createListener: func(_ context.Context, input *CreateListenerInput) (*Listener, error) {
			if input.Port != 80 {
				t.Errorf("expected listener port 80, got %d", input.Port)
			}
			return &Listener{ARN: "arn:listener:1", Port: 80}, nil
		},
	}

	out, err := EnsureLoadBalancer(context.Background(), mock, &EnsureLoadBalancerInput{
		Name:        "my-alb",
		ServiceName: "my-svc",
		VPCID:       "vpc-123",
		SubnetIDs:   []string{"subnet-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.LoadBalancerARN != "arn:lb:1" {
		t.Errorf("expected LoadBalancerARN arn:lb:1, got %s", out.LoadBalancerARN)
	}
	if out.TargetGroupARN != "arn:tg:1" {
		t.Errorf("expected TargetGroupARN arn:tg:1, got %s", out.TargetGroupARN)
	}
	if out.ListenerARN != "arn:listener:1" {
		t.Errorf("expected ListenerARN arn:listener:1, got %s", out.ListenerARN)
	}
	if out.DNSName != "lb.example.com" {
		t.Errorf("expected DNSName lb.example.com, got %s", out.DNSName)
	}
}

func TestEnsureLoadBalancer_ALBExists(t *testing.T) {
	mock := &mockALBClient{
		describeLoadBalancers: func(_ context.Context, _ []string) ([]LoadBalancer, error) {
			return []LoadBalancer{{ARN: "arn:lb:existing", DNSName: "existing.example.com", Name: "my-alb"}}, nil
		},
		describeTargetGroups: func(_ context.Context, _ []string) ([]TargetGroup, error) {
			return nil, errors.New("not found")
		},
		createTargetGroup: func(_ context.Context, input *CreateTargetGroupInput) (*TargetGroup, error) {
			return &TargetGroup{ARN: "arn:tg:1", Name: input.Name}, nil
		},
		describeListeners: func(_ context.Context, _ string) ([]Listener, error) {
			return nil, errors.New("not found")
		},
		createListener: func(_ context.Context, _ *CreateListenerInput) (*Listener, error) {
			return &Listener{ARN: "arn:listener:1", Port: 80}, nil
		},
	}

	out, err := EnsureLoadBalancer(context.Background(), mock, &EnsureLoadBalancerInput{
		Name:        "my-alb",
		ServiceName: "my-svc",
		VPCID:       "vpc-123",
		SubnetIDs:   []string{"subnet-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.LoadBalancerARN != "arn:lb:existing" {
		t.Errorf("expected reused ALB ARN, got %s", out.LoadBalancerARN)
	}
	if out.DNSName != "existing.example.com" {
		t.Errorf("expected reused DNS name, got %s", out.DNSName)
	}
}

func TestEnsureLoadBalancer_TargetGroupExists(t *testing.T) {
	mock := &mockALBClient{
		describeLoadBalancers: func(_ context.Context, _ []string) ([]LoadBalancer, error) {
			return []LoadBalancer{{ARN: "arn:lb:1", DNSName: "lb.example.com"}}, nil
		},
		describeTargetGroups: func(_ context.Context, _ []string) ([]TargetGroup, error) {
			return []TargetGroup{{ARN: "arn:tg:existing", Name: "my-svc-tg"}}, nil
		},
		describeListeners: func(_ context.Context, _ string) ([]Listener, error) {
			return nil, errors.New("not found")
		},
		createListener: func(_ context.Context, _ *CreateListenerInput) (*Listener, error) {
			return &Listener{ARN: "arn:listener:1", Port: 80}, nil
		},
	}

	out, err := EnsureLoadBalancer(context.Background(), mock, &EnsureLoadBalancerInput{
		Name:        "my-alb",
		ServiceName: "my-svc",
		VPCID:       "vpc-123",
		SubnetIDs:   []string{"subnet-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.TargetGroupARN != "arn:tg:existing" {
		t.Errorf("expected reused target group ARN, got %s", out.TargetGroupARN)
	}
}

func TestEnsureLoadBalancer_ListenerExists(t *testing.T) {
	mock := &mockALBClient{
		describeLoadBalancers: func(_ context.Context, _ []string) ([]LoadBalancer, error) {
			return []LoadBalancer{{ARN: "arn:lb:1", DNSName: "lb.example.com"}}, nil
		},
		describeTargetGroups: func(_ context.Context, _ []string) ([]TargetGroup, error) {
			return []TargetGroup{{ARN: "arn:tg:1", Name: "my-svc-tg"}}, nil
		},
		describeListeners: func(_ context.Context, _ string) ([]Listener, error) {
			return []Listener{{ARN: "arn:listener:existing", Port: 80}}, nil
		},
	}

	out, err := EnsureLoadBalancer(context.Background(), mock, &EnsureLoadBalancerInput{
		Name:        "my-alb",
		ServiceName: "my-svc",
		VPCID:       "vpc-123",
		SubnetIDs:   []string{"subnet-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ListenerARN != "arn:listener:existing" {
		t.Errorf("expected reused listener ARN, got %s", out.ListenerARN)
	}
}

func TestEnsureLoadBalancer_CreateALBFails(t *testing.T) {
	mock := &mockALBClient{
		describeLoadBalancers: func(_ context.Context, _ []string) ([]LoadBalancer, error) {
			return nil, errors.New("not found")
		},
		createLoadBalancer: func(_ context.Context, _ *CreateLoadBalancerInput) (*LoadBalancer, error) {
			return nil, errors.New("access denied")
		},
	}

	_, err := EnsureLoadBalancer(context.Background(), mock, &EnsureLoadBalancerInput{
		Name:        "my-alb",
		ServiceName: "my-svc",
		VPCID:       "vpc-123",
		SubnetIDs:   []string{"subnet-1"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errors.Unwrap(err)) && err.Error() != "create load balancer: access denied" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEnsureLoadBalancer_CreateTargetGroupFails(t *testing.T) {
	mock := &mockALBClient{
		describeLoadBalancers: func(_ context.Context, _ []string) ([]LoadBalancer, error) {
			return []LoadBalancer{{ARN: "arn:lb:1", DNSName: "lb.example.com"}}, nil
		},
		describeTargetGroups: func(_ context.Context, _ []string) ([]TargetGroup, error) {
			return nil, errors.New("not found")
		},
		createTargetGroup: func(_ context.Context, _ *CreateTargetGroupInput) (*TargetGroup, error) {
			return nil, errors.New("limit exceeded")
		},
	}

	_, err := EnsureLoadBalancer(context.Background(), mock, &EnsureLoadBalancerInput{
		Name:        "my-alb",
		ServiceName: "my-svc",
		VPCID:       "vpc-123",
		SubnetIDs:   []string{"subnet-1"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "create target group: limit exceeded" {
		t.Errorf("unexpected error message: %v", err)
	}
}
