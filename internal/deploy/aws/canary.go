package aws

import (
	"context"
	"errors"
	"fmt"
)

// CanaryClient abstracts ALB operations needed for canary traffic splitting.
type CanaryClient interface {
	DescribeTargetGroups(ctx context.Context, names []string) ([]TargetGroup, error)
	CreateTargetGroup(ctx context.Context, input *CreateTargetGroupInput) (*TargetGroup, error)
	DescribeListeners(ctx context.Context, loadBalancerARN string) ([]Listener, error)
	ModifyListener(ctx context.Context, input *ModifyListenerInput) error
	RegisterTargets(ctx context.Context, targetGroupARN string, targets []Target) error
}

// CanaryConfig describes a canary deployment via ALB weighted target groups.
type CanaryConfig struct {
	LoadBalancerARN   string
	StableTargetGroup string // ARN of the existing stable target group
	ServiceName       string
	VPCID             string
	Port              int
	CanaryPercent     int // 1-99
	HealthCheckPath   string
}

// CanaryResult describes the outcome of a canary traffic split.
type CanaryResult struct {
	CanaryTargetGroupARN string
	StableTargetGroupARN string
	CanaryPercent        int
	StablePercent        int
}

// SetCanaryTraffic creates a canary target group and configures the ALB
// listener with weighted forward actions splitting traffic between the stable
// and canary target groups.
func SetCanaryTraffic(ctx context.Context, client CanaryClient, config CanaryConfig) (*CanaryResult, error) {
	if err := validateCanaryConfig(config); err != nil {
		return nil, err
	}

	// Create canary target group.
	canaryName := config.ServiceName + "-canary"
	tg, err := client.CreateTargetGroup(ctx, &CreateTargetGroupInput{
		Name:            canaryName,
		VPCID:           config.VPCID,
		Port:            config.Port,
		Protocol:        "HTTP",
		TargetType:      "ip",
		HealthCheckPath: config.HealthCheckPath,
	})
	if err != nil {
		return nil, fmt.Errorf("create canary target group: %w", err)
	}

	// Find listener on the ALB.
	listeners, err := client.DescribeListeners(ctx, config.LoadBalancerARN)
	if err != nil {
		return nil, fmt.Errorf("describe listeners: %w", err)
	}
	if len(listeners) == 0 {
		return nil, errors.New("no listeners found on load balancer")
	}

	// Modify the first listener with weighted forward actions.
	stablePercent := 100 - config.CanaryPercent
	err = client.ModifyListener(ctx, &ModifyListenerInput{
		ListenerARN: listeners[0].ARN,
		Actions: []ListenerAction{
			{Type: "forward", TargetGroupARN: config.StableTargetGroup, Weight: stablePercent},
			{Type: "forward", TargetGroupARN: tg.ARN, Weight: config.CanaryPercent},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("modify listener for canary split: %w", err)
	}

	return &CanaryResult{
		CanaryTargetGroupARN: tg.ARN,
		StableTargetGroupARN: config.StableTargetGroup,
		CanaryPercent:        config.CanaryPercent,
		StablePercent:        stablePercent,
	}, nil
}

// PromoteCanary shifts 100% of traffic to the canary target group by updating
// the ALB listener to forward all traffic to the canary.
func PromoteCanary(ctx context.Context, client CanaryClient, loadBalancerARN, canaryTargetGroupARN string) error {
	if loadBalancerARN == "" {
		return errors.New("load balancer ARN must not be empty")
	}
	if canaryTargetGroupARN == "" {
		return errors.New("canary target group ARN must not be empty")
	}

	listeners, err := client.DescribeListeners(ctx, loadBalancerARN)
	if err != nil {
		return fmt.Errorf("describe listeners: %w", err)
	}
	if len(listeners) == 0 {
		return errors.New("no listeners found on load balancer")
	}

	err = client.ModifyListener(ctx, &ModifyListenerInput{
		ListenerARN: listeners[0].ARN,
		Actions: []ListenerAction{
			{Type: "forward", TargetGroupARN: canaryTargetGroupARN, Weight: 100},
		},
	})
	if err != nil {
		return fmt.Errorf("promote canary: %w", err)
	}

	return nil
}

func validateCanaryConfig(config CanaryConfig) error {
	if config.LoadBalancerARN == "" {
		return errors.New("load balancer ARN must not be empty")
	}
	if config.StableTargetGroup == "" {
		return errors.New("stable target group ARN must not be empty")
	}
	if config.ServiceName == "" {
		return errors.New("service name must not be empty")
	}
	if config.VPCID == "" {
		return errors.New("VPC ID must not be empty")
	}
	if config.Port == 0 {
		return errors.New("port must not be zero")
	}
	if config.CanaryPercent < 1 || config.CanaryPercent > 99 {
		return fmt.Errorf("canary percent must be between 1 and 99, got %d", config.CanaryPercent)
	}
	return nil
}
