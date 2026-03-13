package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// mockElbv2API is a test double for the elbv2API interface.
type mockElbv2API struct {
	createLoadBalancerFn    func(ctx context.Context, input *elbv2.CreateLoadBalancerInput, opts ...func(*elbv2.Options)) (*elbv2.CreateLoadBalancerOutput, error)
	describeLoadBalancersFn func(ctx context.Context, input *elbv2.DescribeLoadBalancersInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
	createTargetGroupFn     func(ctx context.Context, input *elbv2.CreateTargetGroupInput, opts ...func(*elbv2.Options)) (*elbv2.CreateTargetGroupOutput, error)
	describeTargetGroupsFn  func(ctx context.Context, input *elbv2.DescribeTargetGroupsInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
	createListenerFn        func(ctx context.Context, input *elbv2.CreateListenerInput, opts ...func(*elbv2.Options)) (*elbv2.CreateListenerOutput, error)
	describeListenersFn     func(ctx context.Context, input *elbv2.DescribeListenersInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error)
	modifyListenerFn        func(ctx context.Context, input *elbv2.ModifyListenerInput, opts ...func(*elbv2.Options)) (*elbv2.ModifyListenerOutput, error)
	registerTargetsFn       func(ctx context.Context, input *elbv2.RegisterTargetsInput, opts ...func(*elbv2.Options)) (*elbv2.RegisterTargetsOutput, error)
	describeTargetHealthFn  func(ctx context.Context, input *elbv2.DescribeTargetHealthInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error)
}

func (m *mockElbv2API) CreateLoadBalancer(ctx context.Context, input *elbv2.CreateLoadBalancerInput, opts ...func(*elbv2.Options)) (*elbv2.CreateLoadBalancerOutput, error) {
	return m.createLoadBalancerFn(ctx, input, opts...)
}

func (m *mockElbv2API) DescribeLoadBalancers(ctx context.Context, input *elbv2.DescribeLoadBalancersInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
	return m.describeLoadBalancersFn(ctx, input, opts...)
}

func (m *mockElbv2API) CreateTargetGroup(ctx context.Context, input *elbv2.CreateTargetGroupInput, opts ...func(*elbv2.Options)) (*elbv2.CreateTargetGroupOutput, error) {
	return m.createTargetGroupFn(ctx, input, opts...)
}

func (m *mockElbv2API) DescribeTargetGroups(ctx context.Context, input *elbv2.DescribeTargetGroupsInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
	return m.describeTargetGroupsFn(ctx, input, opts...)
}

func (m *mockElbv2API) CreateListener(ctx context.Context, input *elbv2.CreateListenerInput, opts ...func(*elbv2.Options)) (*elbv2.CreateListenerOutput, error) {
	return m.createListenerFn(ctx, input, opts...)
}

func (m *mockElbv2API) DescribeListeners(ctx context.Context, input *elbv2.DescribeListenersInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
	return m.describeListenersFn(ctx, input, opts...)
}

func (m *mockElbv2API) ModifyListener(ctx context.Context, input *elbv2.ModifyListenerInput, opts ...func(*elbv2.Options)) (*elbv2.ModifyListenerOutput, error) {
	return m.modifyListenerFn(ctx, input, opts...)
}

func (m *mockElbv2API) RegisterTargets(ctx context.Context, input *elbv2.RegisterTargetsInput, opts ...func(*elbv2.Options)) (*elbv2.RegisterTargetsOutput, error) {
	return m.registerTargetsFn(ctx, input, opts...)
}

func (m *mockElbv2API) DescribeTargetHealth(ctx context.Context, input *elbv2.DescribeTargetHealthInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
	return m.describeTargetHealthFn(ctx, input, opts...)
}

// ---------------------------------------------------------------------------
// NewALBAdapter
// ---------------------------------------------------------------------------

func TestNewALBAdapter(t *testing.T) {
	adapter := NewALBAdapter(aws.Config{Region: "us-west-2"})
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.client == nil {
		t.Fatal("expected non-nil client")
	}
}

// ---------------------------------------------------------------------------
// CreateLoadBalancer
// ---------------------------------------------------------------------------

func TestALBAdapter_CreateLoadBalancer(t *testing.T) {
	tests := []struct {
		name    string
		input   *CreateLoadBalancerInput
		mock    func(ctx context.Context, in *elbv2.CreateLoadBalancerInput, opts ...func(*elbv2.Options)) (*elbv2.CreateLoadBalancerOutput, error)
		wantLB  *LoadBalancer
		wantErr bool
	}{
		{
			name: "success internet-facing",
			input: &CreateLoadBalancerInput{
				Name:             "my-alb",
				SubnetIDs:        []string{"subnet-1", "subnet-2"},
				SecurityGroupIDs: []string{"sg-1"},
				Internal:         false,
			},
			mock: func(_ context.Context, in *elbv2.CreateLoadBalancerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateLoadBalancerOutput, error) {
				if in.Scheme != elbv2types.LoadBalancerSchemeEnumInternetFacing {
					return nil, errors.New("wrong scheme")
				}
				return &elbv2.CreateLoadBalancerOutput{
					LoadBalancers: []elbv2types.LoadBalancer{
						{
							LoadBalancerArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc"),
							DNSName:          aws.String("my-alb-123.us-east-1.elb.amazonaws.com"),
							LoadBalancerName: aws.String("my-alb"),
						},
					},
				}, nil
			},
			wantLB: &LoadBalancer{
				ARN:     "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc",
				DNSName: "my-alb-123.us-east-1.elb.amazonaws.com",
				Name:    "my-alb",
			},
		},
		{
			name: "success internal",
			input: &CreateLoadBalancerInput{
				Name:     "internal-alb",
				Internal: true,
			},
			mock: func(_ context.Context, in *elbv2.CreateLoadBalancerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateLoadBalancerOutput, error) {
				if in.Scheme != elbv2types.LoadBalancerSchemeEnumInternal {
					return nil, errors.New("expected internal scheme")
				}
				return &elbv2.CreateLoadBalancerOutput{
					LoadBalancers: []elbv2types.LoadBalancer{
						{
							LoadBalancerArn:  aws.String("arn:lb:internal"),
							DNSName:          aws.String("internal.elb"),
							LoadBalancerName: aws.String("internal-alb"),
						},
					},
				}, nil
			},
			wantLB: &LoadBalancer{ARN: "arn:lb:internal", DNSName: "internal.elb", Name: "internal-alb"},
		},
		{
			name:  "sdk error",
			input: &CreateLoadBalancerInput{Name: "fail-alb"},
			mock: func(_ context.Context, _ *elbv2.CreateLoadBalancerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateLoadBalancerOutput, error) {
				return nil, errors.New("access denied")
			},
			wantErr: true,
		},
		{
			name:  "empty result",
			input: &CreateLoadBalancerInput{Name: "empty-alb"},
			mock: func(_ context.Context, _ *elbv2.CreateLoadBalancerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateLoadBalancerOutput, error) {
				return &elbv2.CreateLoadBalancerOutput{LoadBalancers: []elbv2types.LoadBalancer{}}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{createLoadBalancerFn: tt.mock}}
			got, err := adapter.CreateLoadBalancer(context.Background(), tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ARN != tt.wantLB.ARN {
				t.Errorf("ARN = %q, want %q", got.ARN, tt.wantLB.ARN)
			}
			if got.DNSName != tt.wantLB.DNSName {
				t.Errorf("DNSName = %q, want %q", got.DNSName, tt.wantLB.DNSName)
			}
			if got.Name != tt.wantLB.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantLB.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DescribeLoadBalancers
// ---------------------------------------------------------------------------

func TestALBAdapter_DescribeLoadBalancers(t *testing.T) {
	tests := []struct {
		name    string
		names   []string
		mock    func(ctx context.Context, in *elbv2.DescribeLoadBalancersInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error)
		wantLen int
		wantErr bool
	}{
		{
			name:  "success multiple",
			names: []string{"alb-1", "alb-2"},
			mock: func(_ context.Context, in *elbv2.DescribeLoadBalancersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
				if len(in.Names) != 2 {
					return nil, errors.New("expected 2 names")
				}
				return &elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: []elbv2types.LoadBalancer{
						{LoadBalancerArn: aws.String("arn:1"), DNSName: aws.String("dns-1"), LoadBalancerName: aws.String("alb-1")},
						{LoadBalancerArn: aws.String("arn:2"), DNSName: aws.String("dns-2"), LoadBalancerName: aws.String("alb-2")},
					},
				}, nil
			},
			wantLen: 2,
		},
		{
			name:  "empty result",
			names: []string{"nonexistent"},
			mock: func(_ context.Context, _ *elbv2.DescribeLoadBalancersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
				return &elbv2.DescribeLoadBalancersOutput{LoadBalancers: []elbv2types.LoadBalancer{}}, nil
			},
			wantLen: 0,
		},
		{
			name:  "nil pointers in response",
			names: []string{"nil-alb"},
			mock: func(_ context.Context, _ *elbv2.DescribeLoadBalancersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
				return &elbv2.DescribeLoadBalancersOutput{
					LoadBalancers: []elbv2types.LoadBalancer{
						{LoadBalancerArn: nil, DNSName: nil, LoadBalancerName: nil},
					},
				}, nil
			},
			wantLen: 1,
		},
		{
			name:  "sdk error",
			names: []string{"fail"},
			mock: func(_ context.Context, _ *elbv2.DescribeLoadBalancersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeLoadBalancersOutput, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{describeLoadBalancersFn: tt.mock}}
			got, err := adapter.DescribeLoadBalancers(context.Background(), tt.names)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreateTargetGroup
// ---------------------------------------------------------------------------

func TestALBAdapter_CreateTargetGroup(t *testing.T) {
	tests := []struct {
		name    string
		input   *CreateTargetGroupInput
		mock    func(ctx context.Context, in *elbv2.CreateTargetGroupInput, opts ...func(*elbv2.Options)) (*elbv2.CreateTargetGroupOutput, error)
		wantTG  *TargetGroup
		wantErr bool
	}{
		{
			name: "success HTTP ip",
			input: &CreateTargetGroupInput{
				Name:            "my-tg",
				VPCID:           "vpc-123",
				Port:            8080,
				Protocol:        "HTTP",
				TargetType:      "ip",
				HealthCheckPath: "/health",
			},
			mock: func(_ context.Context, in *elbv2.CreateTargetGroupInput, _ ...func(*elbv2.Options)) (*elbv2.CreateTargetGroupOutput, error) {
				if in.Protocol != elbv2types.ProtocolEnumHttp {
					return nil, errors.New("expected HTTP protocol")
				}
				if in.TargetType != elbv2types.TargetTypeEnumIp {
					return nil, errors.New("expected ip target type")
				}
				if in.HealthCheckPath == nil || *in.HealthCheckPath != "/health" {
					return nil, errors.New("expected health check path")
				}
				return &elbv2.CreateTargetGroupOutput{
					TargetGroups: []elbv2types.TargetGroup{
						{TargetGroupArn: aws.String("arn:tg:1"), TargetGroupName: aws.String("my-tg")},
					},
				}, nil
			},
			wantTG: &TargetGroup{ARN: "arn:tg:1", Name: "my-tg"},
		},
		{
			name: "success HTTPS instance",
			input: &CreateTargetGroupInput{
				Name:       "https-tg",
				VPCID:      "vpc-456",
				Port:       443,
				Protocol:   "HTTPS",
				TargetType: "instance",
			},
			mock: func(_ context.Context, in *elbv2.CreateTargetGroupInput, _ ...func(*elbv2.Options)) (*elbv2.CreateTargetGroupOutput, error) {
				if in.Protocol != elbv2types.ProtocolEnumHttps {
					return nil, errors.New("expected HTTPS protocol")
				}
				if in.TargetType != elbv2types.TargetTypeEnumInstance {
					return nil, errors.New("expected instance target type")
				}
				if in.HealthCheckPath != nil {
					return nil, errors.New("health check path should be nil when empty")
				}
				return &elbv2.CreateTargetGroupOutput{
					TargetGroups: []elbv2types.TargetGroup{
						{TargetGroupArn: aws.String("arn:tg:https"), TargetGroupName: aws.String("https-tg")},
					},
				}, nil
			},
			wantTG: &TargetGroup{ARN: "arn:tg:https", Name: "https-tg"},
		},
		{
			name:  "sdk error",
			input: &CreateTargetGroupInput{Name: "fail-tg", VPCID: "vpc-1", Port: 80},
			mock: func(_ context.Context, _ *elbv2.CreateTargetGroupInput, _ ...func(*elbv2.Options)) (*elbv2.CreateTargetGroupOutput, error) {
				return nil, errors.New("limit exceeded")
			},
			wantErr: true,
		},
		{
			name:  "empty result",
			input: &CreateTargetGroupInput{Name: "empty-tg", VPCID: "vpc-1", Port: 80},
			mock: func(_ context.Context, _ *elbv2.CreateTargetGroupInput, _ ...func(*elbv2.Options)) (*elbv2.CreateTargetGroupOutput, error) {
				return &elbv2.CreateTargetGroupOutput{TargetGroups: []elbv2types.TargetGroup{}}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{createTargetGroupFn: tt.mock}}
			got, err := adapter.CreateTargetGroup(context.Background(), tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ARN != tt.wantTG.ARN {
				t.Errorf("ARN = %q, want %q", got.ARN, tt.wantTG.ARN)
			}
			if got.Name != tt.wantTG.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantTG.Name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DescribeTargetGroups
// ---------------------------------------------------------------------------

func TestALBAdapter_DescribeTargetGroups(t *testing.T) {
	tests := []struct {
		name    string
		names   []string
		mock    func(ctx context.Context, in *elbv2.DescribeTargetGroupsInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
		wantLen int
		wantErr bool
	}{
		{
			name:  "success",
			names: []string{"tg-1"},
			mock: func(_ context.Context, _ *elbv2.DescribeTargetGroupsInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
				return &elbv2.DescribeTargetGroupsOutput{
					TargetGroups: []elbv2types.TargetGroup{
						{TargetGroupArn: aws.String("arn:tg:1"), TargetGroupName: aws.String("tg-1")},
					},
				}, nil
			},
			wantLen: 1,
		},
		{
			name:  "empty",
			names: []string{"nonexistent"},
			mock: func(_ context.Context, _ *elbv2.DescribeTargetGroupsInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
				return &elbv2.DescribeTargetGroupsOutput{TargetGroups: []elbv2types.TargetGroup{}}, nil
			},
			wantLen: 0,
		},
		{
			name:  "nil pointers",
			names: []string{"nil-tg"},
			mock: func(_ context.Context, _ *elbv2.DescribeTargetGroupsInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
				return &elbv2.DescribeTargetGroupsOutput{
					TargetGroups: []elbv2types.TargetGroup{
						{TargetGroupArn: nil, TargetGroupName: nil},
					},
				}, nil
			},
			wantLen: 1,
		},
		{
			name:  "error",
			names: []string{"fail"},
			mock: func(_ context.Context, _ *elbv2.DescribeTargetGroupsInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{describeTargetGroupsFn: tt.mock}}
			got, err := adapter.DescribeTargetGroups(context.Background(), tt.names)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreateListener
// ---------------------------------------------------------------------------

func TestALBAdapter_CreateListener(t *testing.T) {
	tests := []struct {
		name    string
		input   *CreateListenerInput
		mock    func(ctx context.Context, in *elbv2.CreateListenerInput, opts ...func(*elbv2.Options)) (*elbv2.CreateListenerOutput, error)
		wantL   *Listener
		wantErr bool
	}{
		{
			name: "success HTTP",
			input: &CreateListenerInput{
				LoadBalancerARN: "arn:lb:1",
				Port:            80,
				Protocol:        "HTTP",
				DefaultActions: []ListenerAction{
					{Type: "forward", TargetGroupARN: "arn:tg:1"},
				},
			},
			mock: func(_ context.Context, in *elbv2.CreateListenerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateListenerOutput, error) {
				if in.Protocol != elbv2types.ProtocolEnumHttp {
					return nil, errors.New("expected HTTP")
				}
				if len(in.DefaultActions) != 1 {
					return nil, errors.New("expected 1 action")
				}
				return &elbv2.CreateListenerOutput{
					Listeners: []elbv2types.Listener{
						{ListenerArn: aws.String("arn:listener:1"), Port: aws.Int32(80)},
					},
				}, nil
			},
			wantL: &Listener{ARN: "arn:listener:1", Port: 80},
		},
		{
			name: "success HTTPS",
			input: &CreateListenerInput{
				LoadBalancerARN: "arn:lb:1",
				Port:            443,
				Protocol:        "HTTPS",
				DefaultActions: []ListenerAction{
					{Type: "forward", TargetGroupARN: "arn:tg:1"},
				},
			},
			mock: func(_ context.Context, in *elbv2.CreateListenerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateListenerOutput, error) {
				if in.Protocol != elbv2types.ProtocolEnumHttps {
					return nil, errors.New("expected HTTPS")
				}
				return &elbv2.CreateListenerOutput{
					Listeners: []elbv2types.Listener{
						{ListenerArn: aws.String("arn:listener:https"), Port: aws.Int32(443)},
					},
				}, nil
			},
			wantL: &Listener{ARN: "arn:listener:https", Port: 443},
		},
		{
			name: "nil port in response",
			input: &CreateListenerInput{
				LoadBalancerARN: "arn:lb:1",
				Port:            80,
				Protocol:        "HTTP",
				DefaultActions:  []ListenerAction{{Type: "forward", TargetGroupARN: "arn:tg:1"}},
			},
			mock: func(_ context.Context, _ *elbv2.CreateListenerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateListenerOutput, error) {
				return &elbv2.CreateListenerOutput{
					Listeners: []elbv2types.Listener{
						{ListenerArn: aws.String("arn:listener:noport"), Port: nil},
					},
				}, nil
			},
			wantL: &Listener{ARN: "arn:listener:noport", Port: 0},
		},
		{
			name:  "sdk error",
			input: &CreateListenerInput{LoadBalancerARN: "arn:lb:1", Port: 80, Protocol: "HTTP"},
			mock: func(_ context.Context, _ *elbv2.CreateListenerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateListenerOutput, error) {
				return nil, errors.New("quota exceeded")
			},
			wantErr: true,
		},
		{
			name:  "empty result",
			input: &CreateListenerInput{LoadBalancerARN: "arn:lb:1", Port: 80, Protocol: "HTTP"},
			mock: func(_ context.Context, _ *elbv2.CreateListenerInput, _ ...func(*elbv2.Options)) (*elbv2.CreateListenerOutput, error) {
				return &elbv2.CreateListenerOutput{Listeners: []elbv2types.Listener{}}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{createListenerFn: tt.mock}}
			got, err := adapter.CreateListener(context.Background(), tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ARN != tt.wantL.ARN {
				t.Errorf("ARN = %q, want %q", got.ARN, tt.wantL.ARN)
			}
			if got.Port != tt.wantL.Port {
				t.Errorf("Port = %d, want %d", got.Port, tt.wantL.Port)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DescribeListeners
// ---------------------------------------------------------------------------

func TestALBAdapter_DescribeListeners(t *testing.T) {
	tests := []struct {
		name    string
		lbARN   string
		mock    func(ctx context.Context, in *elbv2.DescribeListenersInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error)
		wantLen int
		wantErr bool
	}{
		{
			name:  "success",
			lbARN: "arn:lb:1",
			mock: func(_ context.Context, in *elbv2.DescribeListenersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
				if *in.LoadBalancerArn != "arn:lb:1" {
					return nil, errors.New("wrong ARN")
				}
				return &elbv2.DescribeListenersOutput{
					Listeners: []elbv2types.Listener{
						{ListenerArn: aws.String("arn:l:1"), Port: aws.Int32(80)},
						{ListenerArn: aws.String("arn:l:2"), Port: aws.Int32(443)},
					},
				}, nil
			},
			wantLen: 2,
		},
		{
			name:  "nil port",
			lbARN: "arn:lb:1",
			mock: func(_ context.Context, _ *elbv2.DescribeListenersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
				return &elbv2.DescribeListenersOutput{
					Listeners: []elbv2types.Listener{
						{ListenerArn: aws.String("arn:l:nilport"), Port: nil},
					},
				}, nil
			},
			wantLen: 1,
		},
		{
			name:  "empty",
			lbARN: "arn:lb:1",
			mock: func(_ context.Context, _ *elbv2.DescribeListenersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
				return &elbv2.DescribeListenersOutput{Listeners: []elbv2types.Listener{}}, nil
			},
			wantLen: 0,
		},
		{
			name:  "error",
			lbARN: "arn:lb:bad",
			mock: func(_ context.Context, _ *elbv2.DescribeListenersInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeListenersOutput, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{describeListenersFn: tt.mock}}
			got, err := adapter.DescribeListeners(context.Background(), tt.lbARN)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ModifyListener
// ---------------------------------------------------------------------------

func TestALBAdapter_ModifyListener(t *testing.T) {
	tests := []struct {
		name    string
		input   *ModifyListenerInput
		mock    func(ctx context.Context, in *elbv2.ModifyListenerInput, opts ...func(*elbv2.Options)) (*elbv2.ModifyListenerOutput, error)
		wantErr bool
	}{
		{
			name: "success",
			input: &ModifyListenerInput{
				ListenerARN: "arn:listener:1",
				Actions:     []ListenerAction{{Type: "forward", TargetGroupARN: "arn:tg:2"}},
			},
			mock: func(_ context.Context, in *elbv2.ModifyListenerInput, _ ...func(*elbv2.Options)) (*elbv2.ModifyListenerOutput, error) {
				if *in.ListenerArn != "arn:listener:1" {
					return nil, errors.New("wrong listener ARN")
				}
				if len(in.DefaultActions) != 1 {
					return nil, errors.New("expected 1 action")
				}
				if string(in.DefaultActions[0].Type) != "forward" {
					return nil, errors.New("expected forward action")
				}
				return &elbv2.ModifyListenerOutput{}, nil
			},
		},
		{
			name: "error",
			input: &ModifyListenerInput{
				ListenerARN: "arn:listener:bad",
				Actions:     []ListenerAction{{Type: "forward", TargetGroupARN: "arn:tg:1"}},
			},
			mock: func(_ context.Context, _ *elbv2.ModifyListenerInput, _ ...func(*elbv2.Options)) (*elbv2.ModifyListenerOutput, error) {
				return nil, errors.New("access denied")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{modifyListenerFn: tt.mock}}
			err := adapter.ModifyListener(context.Background(), tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RegisterTargets
// ---------------------------------------------------------------------------

func TestALBAdapter_RegisterTargets(t *testing.T) {
	tests := []struct {
		name    string
		tgARN   string
		targets []Target
		mock    func(ctx context.Context, in *elbv2.RegisterTargetsInput, opts ...func(*elbv2.Options)) (*elbv2.RegisterTargetsOutput, error)
		wantErr bool
	}{
		{
			name:  "success",
			tgARN: "arn:tg:1",
			targets: []Target{
				{ID: "10.0.0.1", Port: 8080},
				{ID: "10.0.0.2", Port: 8080},
			},
			mock: func(_ context.Context, in *elbv2.RegisterTargetsInput, _ ...func(*elbv2.Options)) (*elbv2.RegisterTargetsOutput, error) {
				if *in.TargetGroupArn != "arn:tg:1" {
					return nil, errors.New("wrong tg ARN")
				}
				if len(in.Targets) != 2 {
					return nil, errors.New("expected 2 targets")
				}
				return &elbv2.RegisterTargetsOutput{}, nil
			},
		},
		{
			name:    "error",
			tgARN:   "arn:tg:bad",
			targets: []Target{{ID: "10.0.0.1", Port: 80}},
			mock: func(_ context.Context, _ *elbv2.RegisterTargetsInput, _ ...func(*elbv2.Options)) (*elbv2.RegisterTargetsOutput, error) {
				return nil, errors.New("invalid target")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{registerTargetsFn: tt.mock}}
			err := adapter.RegisterTargets(context.Background(), tt.tgARN, tt.targets)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DescribeTargetHealth
// ---------------------------------------------------------------------------

func TestALBAdapter_DescribeTargetHealth(t *testing.T) {
	tests := []struct {
		name    string
		tgARN   string
		mock    func(ctx context.Context, in *elbv2.DescribeTargetHealthInput, opts ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error)
		wantLen int
		wantErr bool
	}{
		{
			name:  "success with health",
			tgARN: "arn:tg:1",
			mock: func(_ context.Context, _ *elbv2.DescribeTargetHealthInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
				return &elbv2.DescribeTargetHealthOutput{
					TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
						{
							Target:       &elbv2types.TargetDescription{Id: aws.String("10.0.0.1")},
							TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumHealthy, Description: aws.String("ok")},
						},
						{
							Target:       &elbv2types.TargetDescription{Id: aws.String("10.0.0.2")},
							TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumUnhealthy, Description: aws.String("timeout")},
						},
					},
				}, nil
			},
			wantLen: 2,
		},
		{
			name:  "nil target and health",
			tgARN: "arn:tg:nil",
			mock: func(_ context.Context, _ *elbv2.DescribeTargetHealthInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
				return &elbv2.DescribeTargetHealthOutput{
					TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
						{Target: nil, TargetHealth: nil},
					},
				}, nil
			},
			wantLen: 1,
		},
		{
			name:  "empty",
			tgARN: "arn:tg:empty",
			mock: func(_ context.Context, _ *elbv2.DescribeTargetHealthInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
				return &elbv2.DescribeTargetHealthOutput{TargetHealthDescriptions: []elbv2types.TargetHealthDescription{}}, nil
			},
			wantLen: 0,
		},
		{
			name:  "error",
			tgARN: "arn:tg:bad",
			mock: func(_ context.Context, _ *elbv2.DescribeTargetHealthInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
				return nil, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &ALBAdapter{client: &mockElbv2API{describeTargetHealthFn: tt.mock}}
			got, err := adapter.DescribeTargetHealth(context.Background(), tt.tgARN)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// TestALBAdapter_DescribeTargetHealth_FieldMapping verifies precise field
// mapping including nil-safety for Target and TargetHealth sub-structs.
func TestALBAdapter_DescribeTargetHealth_FieldMapping(t *testing.T) {
	mock := &mockElbv2API{
		describeTargetHealthFn: func(_ context.Context, _ *elbv2.DescribeTargetHealthInput, _ ...func(*elbv2.Options)) (*elbv2.DescribeTargetHealthOutput, error) {
			return &elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
					{
						Target:       &elbv2types.TargetDescription{Id: aws.String("10.0.0.1")},
						TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumHealthy, Description: aws.String("healthy")},
					},
					{
						Target:       nil,
						TargetHealth: &elbv2types.TargetHealth{State: elbv2types.TargetHealthStateEnumUnhealthy, Description: nil},
					},
					{
						Target:       &elbv2types.TargetDescription{Id: aws.String("10.0.0.3")},
						TargetHealth: nil,
					},
				},
			}, nil
		},
	}
	adapter := &ALBAdapter{client: mock}
	got, err := adapter.DescribeTargetHealth(context.Background(), "arn:tg:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}
	// First: all fields populated.
	if got[0].TargetID != "10.0.0.1" || got[0].State != "healthy" || got[0].Description != "healthy" {
		t.Errorf("got[0] = %+v", got[0])
	}
	// Second: nil target, non-nil health with nil description.
	if got[1].TargetID != "" || got[1].State != "unhealthy" || got[1].Description != "" {
		t.Errorf("got[1] = %+v", got[1])
	}
	// Third: non-nil target, nil health.
	if got[2].TargetID != "10.0.0.3" || got[2].State != "" || got[2].Description != "" {
		t.Errorf("got[2] = %+v", got[2])
	}
}

// ---------------------------------------------------------------------------
// EnsureLoadBalancer (uses the domain-level ALBClient mock)
// ---------------------------------------------------------------------------

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
	if err.Error() != "create load balancer: access denied" {
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

func TestEnsureLoadBalancer_CreateListenerFails(t *testing.T) {
	mock := &mockALBClient{
		describeLoadBalancers: func(_ context.Context, _ []string) ([]LoadBalancer, error) {
			return []LoadBalancer{{ARN: "arn:lb:1", DNSName: "lb.example.com"}}, nil
		},
		describeTargetGroups: func(_ context.Context, _ []string) ([]TargetGroup, error) {
			return []TargetGroup{{ARN: "arn:tg:1", Name: "my-svc-tg"}}, nil
		},
		describeListeners: func(_ context.Context, _ string) ([]Listener, error) {
			return nil, errors.New("not found")
		},
		createListener: func(_ context.Context, _ *CreateListenerInput) (*Listener, error) {
			return nil, errors.New("quota exceeded")
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
	if err.Error() != "create listener: quota exceeded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEnsureLoadBalancer_CustomPortAndHealthCheck(t *testing.T) {
	mock := &mockALBClient{
		describeLoadBalancers: func(_ context.Context, _ []string) ([]LoadBalancer, error) {
			return nil, errors.New("not found")
		},
		createLoadBalancer: func(_ context.Context, _ *CreateLoadBalancerInput) (*LoadBalancer, error) {
			return &LoadBalancer{ARN: "arn:lb:1", DNSName: "lb.example.com"}, nil
		},
		describeTargetGroups: func(_ context.Context, _ []string) ([]TargetGroup, error) {
			return nil, errors.New("not found")
		},
		createTargetGroup: func(_ context.Context, input *CreateTargetGroupInput) (*TargetGroup, error) {
			if input.Port != 9090 {
				t.Errorf("expected port 9090, got %d", input.Port)
			}
			if input.HealthCheckPath != "/ready" {
				t.Errorf("expected health check path /ready, got %s", input.HealthCheckPath)
			}
			return &TargetGroup{ARN: "arn:tg:1", Name: input.Name}, nil
		},
		describeListeners: func(_ context.Context, _ string) ([]Listener, error) {
			return nil, errors.New("not found")
		},
		createListener: func(_ context.Context, _ *CreateListenerInput) (*Listener, error) {
			return &Listener{ARN: "arn:listener:1", Port: 80}, nil
		},
	}

	_, err := EnsureLoadBalancer(context.Background(), mock, &EnsureLoadBalancerInput{
		Name:            "my-alb",
		ServiceName:     "my-svc",
		VPCID:           "vpc-123",
		SubnetIDs:       []string{"subnet-1"},
		Port:            9090,
		HealthCheckPath: "/ready",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
