package aws

import (
	"context"
)

// ALBClient abstracts Elastic Load Balancing v2 (Application Load Balancer) operations.
type ALBClient interface {
	// CreateLoadBalancer creates an Application Load Balancer.
	CreateLoadBalancer(ctx context.Context, input *CreateLoadBalancerInput) (*LoadBalancer, error)

	// DescribeLoadBalancers returns metadata for the named load balancers.
	DescribeLoadBalancers(ctx context.Context, names []string) ([]LoadBalancer, error)

	// CreateTargetGroup creates a target group for routing traffic.
	CreateTargetGroup(ctx context.Context, input *CreateTargetGroupInput) (*TargetGroup, error)

	// DescribeTargetGroups returns metadata for the named target groups.
	DescribeTargetGroups(ctx context.Context, names []string) ([]TargetGroup, error)

	// CreateListener creates a listener on a load balancer.
	CreateListener(ctx context.Context, input *CreateListenerInput) (*Listener, error)

	// DescribeListeners returns listeners for a load balancer.
	DescribeListeners(ctx context.Context, loadBalancerARN string) ([]Listener, error)

	// ModifyListener updates the actions on an existing listener.
	ModifyListener(ctx context.Context, input *ModifyListenerInput) error

	// RegisterTargets registers targets with a target group.
	RegisterTargets(ctx context.Context, targetGroupARN string, targets []Target) error

	// DescribeTargetHealth returns the health status of targets in a target group.
	DescribeTargetHealth(ctx context.Context, targetGroupARN string) ([]TargetHealth, error)
}

// CreateLoadBalancerInput holds parameters for creating an ALB.
type CreateLoadBalancerInput struct {
	Name             string
	SubnetIDs        []string
	SecurityGroupIDs []string
	Internal         bool
}

// LoadBalancer represents an Application Load Balancer.
type LoadBalancer struct {
	ARN     string
	DNSName string
	Name    string
}

// CreateTargetGroupInput holds parameters for creating a target group.
type CreateTargetGroupInput struct {
	Name            string
	VPCID           string
	Port            int
	Protocol        string
	TargetType      string
	HealthCheckPath string
}

// TargetGroup represents an ALB target group.
type TargetGroup struct {
	ARN  string
	Name string
}

// CreateListenerInput holds parameters for creating a listener.
type CreateListenerInput struct {
	LoadBalancerARN string
	Port            int
	Protocol        string
	DefaultActions  []ListenerAction
}

// Listener represents an ALB listener.
type Listener struct {
	ARN  string
	Port int
}

// ModifyListenerInput holds parameters for modifying a listener.
type ModifyListenerInput struct {
	ListenerARN string
	Actions     []ListenerAction
}

// ListenerAction describes a routing action for a listener rule.
type ListenerAction struct {
	Type           string
	TargetGroupARN string
	Weight         int
}

// Target identifies a target for registration with a target group.
type Target struct {
	ID   string
	Port int
}

// TargetHealth describes the health of a registered target.
type TargetHealth struct {
	TargetID    string
	State       string
	Description string
}
