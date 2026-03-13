package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// ALBAdapter implements ALBClient using the AWS SDK v2.
type ALBAdapter struct {
	client *elbv2.Client
}

var _ ALBClient = (*ALBAdapter)(nil)

// NewALBAdapter creates a new adapter backed by the AWS ELBv2 SDK client.
func NewALBAdapter(cfg aws.Config) *ALBAdapter {
	return &ALBAdapter{client: elbv2.NewFromConfig(cfg)}
}

// CreateLoadBalancer creates an Application Load Balancer.
func (a *ALBAdapter) CreateLoadBalancer(ctx context.Context, input *CreateLoadBalancerInput) (*LoadBalancer, error) {
	scheme := elbv2types.LoadBalancerSchemeEnumInternetFacing
	if input.Internal {
		scheme = elbv2types.LoadBalancerSchemeEnumInternal
	}
	out, err := a.client.CreateLoadBalancer(ctx, &elbv2.CreateLoadBalancerInput{
		Name:           &input.Name,
		Subnets:        input.SubnetIDs,
		SecurityGroups: input.SecurityGroupIDs,
		Scheme:         scheme,
		Type:           elbv2types.LoadBalancerTypeEnumApplication,
	})
	if err != nil {
		return nil, err
	}
	if len(out.LoadBalancers) == 0 {
		return nil, fmt.Errorf("alb: no load balancer returned after creation")
	}
	lb := out.LoadBalancers[0]
	return &LoadBalancer{
		ARN:     derefStr(lb.LoadBalancerArn),
		DNSName: derefStr(lb.DNSName),
		Name:    derefStr(lb.LoadBalancerName),
	}, nil
}

// DescribeLoadBalancers returns metadata for the named load balancers.
func (a *ALBAdapter) DescribeLoadBalancers(ctx context.Context, names []string) ([]LoadBalancer, error) {
	out, err := a.client.DescribeLoadBalancers(ctx, &elbv2.DescribeLoadBalancersInput{
		Names: names,
	})
	if err != nil {
		return nil, err
	}
	lbs := make([]LoadBalancer, len(out.LoadBalancers))
	for i, lb := range out.LoadBalancers {
		lbs[i] = LoadBalancer{
			ARN:     derefStr(lb.LoadBalancerArn),
			DNSName: derefStr(lb.DNSName),
			Name:    derefStr(lb.LoadBalancerName),
		}
	}
	return lbs, nil
}

// CreateTargetGroup creates a target group for routing traffic.
func (a *ALBAdapter) CreateTargetGroup(ctx context.Context, input *CreateTargetGroupInput) (*TargetGroup, error) {
	protocol := elbv2types.ProtocolEnumHttp
	if input.Protocol == "HTTPS" {
		protocol = elbv2types.ProtocolEnumHttps
	}
	targetType := elbv2types.TargetTypeEnumIp
	if input.TargetType == "instance" {
		targetType = elbv2types.TargetTypeEnumInstance
	}
	sdkInput := &elbv2.CreateTargetGroupInput{
		Name:       &input.Name,
		VpcId:      &input.VPCID,
		Port:       aws.Int32(int32(input.Port)),
		Protocol:   protocol,
		TargetType: targetType,
	}
	if input.HealthCheckPath != "" {
		sdkInput.HealthCheckPath = &input.HealthCheckPath
	}
	out, err := a.client.CreateTargetGroup(ctx, sdkInput)
	if err != nil {
		return nil, err
	}
	if len(out.TargetGroups) == 0 {
		return nil, fmt.Errorf("alb: no target group returned after creation")
	}
	tg := out.TargetGroups[0]
	return &TargetGroup{
		ARN:  derefStr(tg.TargetGroupArn),
		Name: derefStr(tg.TargetGroupName),
	}, nil
}

// DescribeTargetGroups returns metadata for the named target groups.
func (a *ALBAdapter) DescribeTargetGroups(ctx context.Context, names []string) ([]TargetGroup, error) {
	out, err := a.client.DescribeTargetGroups(ctx, &elbv2.DescribeTargetGroupsInput{
		Names: names,
	})
	if err != nil {
		return nil, err
	}
	tgs := make([]TargetGroup, len(out.TargetGroups))
	for i, tg := range out.TargetGroups {
		tgs[i] = TargetGroup{
			ARN:  derefStr(tg.TargetGroupArn),
			Name: derefStr(tg.TargetGroupName),
		}
	}
	return tgs, nil
}

// CreateListener creates a listener on a load balancer.
func (a *ALBAdapter) CreateListener(ctx context.Context, input *CreateListenerInput) (*Listener, error) {
	protocol := elbv2types.ProtocolEnumHttp
	if input.Protocol == "HTTPS" {
		protocol = elbv2types.ProtocolEnumHttps
	}
	actions := make([]elbv2types.Action, len(input.DefaultActions))
	for i, act := range input.DefaultActions {
		actions[i] = elbv2types.Action{
			Type:           elbv2types.ActionTypeEnum(act.Type),
			TargetGroupArn: &act.TargetGroupARN,
		}
	}
	out, err := a.client.CreateListener(ctx, &elbv2.CreateListenerInput{
		LoadBalancerArn: &input.LoadBalancerARN,
		Port:            aws.Int32(int32(input.Port)),
		Protocol:        protocol,
		DefaultActions:  actions,
	})
	if err != nil {
		return nil, err
	}
	if len(out.Listeners) == 0 {
		return nil, fmt.Errorf("alb: no listener returned after creation")
	}
	l := out.Listeners[0]
	var port int
	if l.Port != nil {
		port = int(*l.Port)
	}
	return &Listener{
		ARN:  derefStr(l.ListenerArn),
		Port: port,
	}, nil
}

// DescribeListeners returns listeners for a load balancer.
func (a *ALBAdapter) DescribeListeners(ctx context.Context, loadBalancerARN string) ([]Listener, error) {
	out, err := a.client.DescribeListeners(ctx, &elbv2.DescribeListenersInput{
		LoadBalancerArn: &loadBalancerARN,
	})
	if err != nil {
		return nil, err
	}
	listeners := make([]Listener, len(out.Listeners))
	for i, l := range out.Listeners {
		var port int
		if l.Port != nil {
			port = int(*l.Port)
		}
		listeners[i] = Listener{
			ARN:  derefStr(l.ListenerArn),
			Port: port,
		}
	}
	return listeners, nil
}

// ModifyListener updates the actions on an existing listener.
func (a *ALBAdapter) ModifyListener(ctx context.Context, input *ModifyListenerInput) error {
	actions := make([]elbv2types.Action, len(input.Actions))
	for i, act := range input.Actions {
		actions[i] = elbv2types.Action{
			Type:           elbv2types.ActionTypeEnum(act.Type),
			TargetGroupArn: &act.TargetGroupARN,
		}
	}
	_, err := a.client.ModifyListener(ctx, &elbv2.ModifyListenerInput{
		ListenerArn:    &input.ListenerARN,
		DefaultActions: actions,
	})
	return err
}

// RegisterTargets registers targets with a target group.
func (a *ALBAdapter) RegisterTargets(ctx context.Context, targetGroupARN string, targets []Target) error {
	descs := make([]elbv2types.TargetDescription, len(targets))
	for i, t := range targets {
		descs[i] = elbv2types.TargetDescription{
			Id:   &t.ID,
			Port: aws.Int32(int32(t.Port)),
		}
	}
	_, err := a.client.RegisterTargets(ctx, &elbv2.RegisterTargetsInput{
		TargetGroupArn: &targetGroupARN,
		Targets:        descs,
	})
	return err
}

// DescribeTargetHealth returns the health status of targets in a target group.
func (a *ALBAdapter) DescribeTargetHealth(ctx context.Context, targetGroupARN string) ([]TargetHealth, error) {
	out, err := a.client.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: &targetGroupARN,
	})
	if err != nil {
		return nil, err
	}
	healths := make([]TargetHealth, len(out.TargetHealthDescriptions))
	for i, h := range out.TargetHealthDescriptions {
		var targetID string
		if h.Target != nil {
			targetID = derefStr(h.Target.Id)
		}
		var state, desc string
		if h.TargetHealth != nil {
			state = string(h.TargetHealth.State)
			desc = derefStr(h.TargetHealth.Description)
		}
		healths[i] = TargetHealth{
			TargetID:    targetID,
			State:       state,
			Description: desc,
		}
	}
	return healths, nil
}

// EnsureLoadBalancerInput holds parameters for ensuring an ALB, target group,
// and listener exist.
type EnsureLoadBalancerInput struct {
	Name             string
	ServiceName      string
	VPCID            string
	SubnetIDs        []string
	SecurityGroupIDs []string
	Internal         bool
	Port             int
	HealthCheckPath  string
}

// EnsureLoadBalancerOutput holds the resources created or found by
// EnsureLoadBalancer.
type EnsureLoadBalancerOutput struct {
	LoadBalancerARN string
	TargetGroupARN  string
	ListenerARN     string
	DNSName         string
}

// EnsureLoadBalancer creates or reuses an ALB, target group, and listener.
func EnsureLoadBalancer(ctx context.Context, client ALBClient, input *EnsureLoadBalancerInput) (*EnsureLoadBalancerOutput, error) {
	// 1. Check if ALB exists.
	var lb LoadBalancer
	lbs, err := client.DescribeLoadBalancers(ctx, []string{input.Name})
	if err != nil || len(lbs) == 0 {
		created, createErr := client.CreateLoadBalancer(ctx, &CreateLoadBalancerInput{
			Name:             input.Name,
			SubnetIDs:        input.SubnetIDs,
			SecurityGroupIDs: input.SecurityGroupIDs,
			Internal:         input.Internal,
		})
		if createErr != nil {
			return nil, fmt.Errorf("create load balancer: %w", createErr)
		}
		lb = *created
	} else {
		lb = lbs[0]
	}

	// 2. Check if target group exists.
	tgName := input.ServiceName + "-tg"
	var tg TargetGroup
	tgs, err := client.DescribeTargetGroups(ctx, []string{tgName})
	if err != nil || len(tgs) == 0 {
		port := input.Port
		if port == 0 {
			port = 80
		}
		healthPath := input.HealthCheckPath
		if healthPath == "" {
			healthPath = "/health"
		}
		created, createErr := client.CreateTargetGroup(ctx, &CreateTargetGroupInput{
			Name:            tgName,
			VPCID:           input.VPCID,
			Port:            port,
			Protocol:        "HTTP",
			TargetType:      "ip",
			HealthCheckPath: healthPath,
		})
		if createErr != nil {
			return nil, fmt.Errorf("create target group: %w", createErr)
		}
		tg = *created
	} else {
		tg = tgs[0]
	}

	// 3. Check if listener exists.
	var listener Listener
	listeners, err := client.DescribeListeners(ctx, lb.ARN)
	if err != nil || len(listeners) == 0 {
		created, createErr := client.CreateListener(ctx, &CreateListenerInput{
			LoadBalancerARN: lb.ARN,
			Port:            80,
			Protocol:        "HTTP",
			DefaultActions: []ListenerAction{
				{
					Type:           "forward",
					TargetGroupARN: tg.ARN,
				},
			},
		})
		if createErr != nil {
			return nil, fmt.Errorf("create listener: %w", createErr)
		}
		listener = *created
	} else {
		listener = listeners[0]
	}

	return &EnsureLoadBalancerOutput{
		LoadBalancerARN: lb.ARN,
		TargetGroupARN:  tg.ARN,
		ListenerARN:     listener.ARN,
		DNSName:         lb.DNSName,
	}, nil
}
