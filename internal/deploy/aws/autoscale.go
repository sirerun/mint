package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	astypes "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
)

const defaultTargetCPUPercent = 70

// autoscaleAPI abstracts the AWS Application Auto Scaling SDK methods used by
// AutoScaleAdapter.
type autoscaleAPI interface {
	RegisterScalableTarget(ctx context.Context, input *applicationautoscaling.RegisterScalableTargetInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.RegisterScalableTargetOutput, error)
	PutScalingPolicy(ctx context.Context, input *applicationautoscaling.PutScalingPolicyInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.PutScalingPolicyOutput, error)
}

// AutoScaleConfig holds parameters for configuring auto-scaling on an ECS service.
type AutoScaleConfig struct {
	ServiceName      string
	ClusterName      string
	MinInstances     int
	MaxInstances     int
	TargetCPUPercent int
}

// AutoScaler configures auto-scaling for an ECS service.
type AutoScaler interface {
	ConfigureAutoScaling(ctx context.Context, config AutoScaleConfig) error
}

// AutoScaleAdapter wraps the AWS Application Auto Scaling SDK client.
type AutoScaleAdapter struct {
	client autoscaleAPI
}

var _ AutoScaler = (*AutoScaleAdapter)(nil)

// NewAutoScaleAdapter creates an AutoScaleAdapter from an AWS config.
func NewAutoScaleAdapter(cfg aws.Config) *AutoScaleAdapter {
	return &AutoScaleAdapter{
		client: applicationautoscaling.NewFromConfig(cfg),
	}
}

// ConfigureAutoScaling registers the ECS service as a scalable target and
// creates a target-tracking scaling policy based on CPU utilization.
func (a *AutoScaleAdapter) ConfigureAutoScaling(ctx context.Context, config AutoScaleConfig) error {
	if config.ServiceName == "" {
		return fmt.Errorf("autoscale: service name must not be empty")
	}
	if config.ClusterName == "" {
		return fmt.Errorf("autoscale: cluster name must not be empty")
	}
	if config.MinInstances < 1 {
		return fmt.Errorf("autoscale: min instances must be at least 1, got %d", config.MinInstances)
	}
	if config.MaxInstances < config.MinInstances {
		return fmt.Errorf("autoscale: max instances (%d) must be >= min instances (%d)", config.MaxInstances, config.MinInstances)
	}

	targetCPU := config.TargetCPUPercent
	if targetCPU == 0 {
		targetCPU = defaultTargetCPUPercent
	}

	resourceID := fmt.Sprintf("service/%s/%s", config.ClusterName, config.ServiceName)

	_, err := a.client.RegisterScalableTarget(ctx, &applicationautoscaling.RegisterScalableTargetInput{
		ServiceNamespace:  astypes.ServiceNamespaceEcs,
		ResourceId:        aws.String(resourceID),
		ScalableDimension: astypes.ScalableDimensionECSServiceDesiredCount,
		MinCapacity:       aws.Int32(int32(config.MinInstances)),
		MaxCapacity:       aws.Int32(int32(config.MaxInstances)),
	})
	if err != nil {
		return fmt.Errorf("autoscale: register scalable target: %w", err)
	}

	policyName := config.ServiceName + "-cpu-target-tracking"
	_, err = a.client.PutScalingPolicy(ctx, &applicationautoscaling.PutScalingPolicyInput{
		ServiceNamespace:  astypes.ServiceNamespaceEcs,
		ResourceId:        aws.String(resourceID),
		ScalableDimension: astypes.ScalableDimensionECSServiceDesiredCount,
		PolicyName:        aws.String(policyName),
		PolicyType:        astypes.PolicyTypeTargetTrackingScaling,
		TargetTrackingScalingPolicyConfiguration: &astypes.TargetTrackingScalingPolicyConfiguration{
			PredefinedMetricSpecification: &astypes.PredefinedMetricSpecification{
				PredefinedMetricType: astypes.MetricTypeECSServiceAverageCPUUtilization,
			},
			TargetValue: aws.Float64(float64(targetCPU)),
		},
	})
	if err != nil {
		return fmt.Errorf("autoscale: put scaling policy: %w", err)
	}

	return nil
}
