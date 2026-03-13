package aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const ecsStableTimeout = 10 * time.Minute

// ECSAdapter wraps the AWS ECS SDK client.
type ECSAdapter struct {
	client *ecs.Client
}

var _ ECSClient = (*ECSAdapter)(nil)

// NewECSAdapter creates an ECSAdapter from an AWS config.
func NewECSAdapter(cfg aws.Config) *ECSAdapter {
	return &ECSAdapter{client: ecs.NewFromConfig(cfg)}
}

func (a *ECSAdapter) CreateCluster(ctx context.Context, clusterName string) (*Cluster, error) {
	out, err := a.client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	if err != nil {
		return nil, fmt.Errorf("ecs: create cluster: %w", err)
	}
	return &Cluster{
		ClusterARN:  aws.ToString(out.Cluster.ClusterArn),
		ClusterName: aws.ToString(out.Cluster.ClusterName),
	}, nil
}

func (a *ECSAdapter) DescribeServices(ctx context.Context, input *DescribeServicesInput) ([]ECSService, error) {
	out, err := a.client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(input.Cluster),
		Services: []string{input.ServiceName},
	})
	if err != nil {
		return nil, fmt.Errorf("ecs: describe services: %w", err)
	}
	var services []ECSService
	for _, s := range out.Services {
		if aws.ToString(s.Status) == "INACTIVE" {
			continue
		}
		services = append(services, ecsServiceFromSDK(s))
	}
	if len(services) == 0 {
		return nil, ErrServiceNotFound
	}
	return services, nil
}

func (a *ECSAdapter) RegisterTaskDefinition(ctx context.Context, input *RegisterTaskDefinitionInput) (*TaskDefinition, error) {
	containerDef := ecstypes.ContainerDefinition{
		Name:  aws.String(input.ContainerName),
		Image: aws.String(input.ImageURI),
		PortMappings: []ecstypes.PortMapping{{
			ContainerPort: aws.Int32(int32(input.Port)),
			Protocol:      ecstypes.TransportProtocolTcp,
		}},
		Essential: aws.Bool(true),
	}
	for k, v := range input.EnvVars {
		containerDef.Environment = append(containerDef.Environment, ecstypes.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	for _, arn := range input.SecretARNs {
		containerDef.Secrets = append(containerDef.Secrets, ecstypes.Secret{
			Name:      aws.String(arn),
			ValueFrom: aws.String(arn),
		})
	}

	sdkInput := &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String(input.Family),
		ContainerDefinitions:    []ecstypes.ContainerDefinition{containerDef},
		Cpu:                     aws.String(input.CPU),
		Memory:                  aws.String(input.Memory),
		NetworkMode:             ecstypes.NetworkModeAwsvpc,
		RequiresCompatibilities: []ecstypes.Compatibility{ecstypes.CompatibilityFargate},
		ExecutionRoleArn:        aws.String(input.ExecutionRoleARN),
	}
	if input.TaskRoleARN != "" {
		sdkInput.TaskRoleArn = aws.String(input.TaskRoleARN)
	}

	out, err := a.client.RegisterTaskDefinition(ctx, sdkInput)
	if err != nil {
		return nil, fmt.Errorf("ecs: register task definition: %w", err)
	}
	return &TaskDefinition{
		TaskDefinitionARN: aws.ToString(out.TaskDefinition.TaskDefinitionArn),
		Family:            aws.ToString(out.TaskDefinition.Family),
		Revision:          int(out.TaskDefinition.Revision),
	}, nil
}

func (a *ECSAdapter) CreateService(ctx context.Context, input *CreateECSServiceInput) (*ECSService, error) {
	assignPublicIP := ecstypes.AssignPublicIpDisabled
	if input.AssignPublicIP {
		assignPublicIP = ecstypes.AssignPublicIpEnabled
	}

	sdkInput := &ecs.CreateServiceInput{
		Cluster:        aws.String(input.Cluster),
		ServiceName:    aws.String(input.ServiceName),
		TaskDefinition: aws.String(input.TaskDefinitionARN),
		DesiredCount:   aws.Int32(int32(input.DesiredCount)),
		LaunchType:     ecstypes.LaunchTypeFargate,
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        input.SubnetIDs,
				SecurityGroups: input.SecurityGroupIDs,
				AssignPublicIp: assignPublicIP,
			},
		},
	}
	if input.TargetGroupARN != "" {
		sdkInput.LoadBalancers = []ecstypes.LoadBalancer{{
			TargetGroupArn: aws.String(input.TargetGroupARN),
			ContainerName:  aws.String(input.ServiceName),
			ContainerPort:  aws.Int32(80),
		}}
	}

	out, err := a.client.CreateService(ctx, sdkInput)
	if err != nil {
		return nil, fmt.Errorf("ecs: create service: %w", err)
	}
	svc := ecsServiceFromSDK(*out.Service)
	return &svc, nil
}

func (a *ECSAdapter) UpdateService(ctx context.Context, input *UpdateECSServiceInput) (*ECSService, error) {
	out, err := a.client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:        aws.String(input.Cluster),
		Service:        aws.String(input.ServiceName),
		TaskDefinition: aws.String(input.TaskDefinitionARN),
		DesiredCount:   aws.Int32(int32(input.DesiredCount)),
	})
	if err != nil {
		return nil, fmt.Errorf("ecs: update service: %w", err)
	}
	svc := ecsServiceFromSDK(*out.Service)
	return &svc, nil
}

func (a *ECSAdapter) DescribeTasks(ctx context.Context, cluster string, taskARNs []string) ([]Task, error) {
	out, err := a.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   taskARNs,
	})
	if err != nil {
		return nil, fmt.Errorf("ecs: describe tasks: %w", err)
	}
	tasks := make([]Task, 0, len(out.Tasks))
	for _, t := range out.Tasks {
		tasks = append(tasks, Task{
			TaskARN:    aws.ToString(t.TaskArn),
			LastStatus: aws.ToString(t.LastStatus),
		})
	}
	return tasks, nil
}

func (a *ECSAdapter) ListTaskDefinitions(ctx context.Context, family string) ([]string, error) {
	out, err := a.client.ListTaskDefinitions(ctx, &ecs.ListTaskDefinitionsInput{
		FamilyPrefix: aws.String(family),
		Sort:         ecstypes.SortOrderDesc,
	})
	if err != nil {
		return nil, fmt.Errorf("ecs: list task definitions: %w", err)
	}
	return out.TaskDefinitionArns, nil
}

func (a *ECSAdapter) WaitForStableService(ctx context.Context, cluster, serviceName string) error {
	waiter := ecs.NewServicesStableWaiter(a.client)
	err := waiter.Wait(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: []string{serviceName},
	}, ecsStableTimeout)
	if err != nil {
		return fmt.Errorf("ecs: wait for stable service: %w", err)
	}
	return nil
}

func ecsServiceFromSDK(s ecstypes.Service) ECSService {
	return ECSService{
		ServiceARN:        aws.ToString(s.ServiceArn),
		ServiceName:       aws.ToString(s.ServiceName),
		Status:            aws.ToString(s.Status),
		ClusterARN:        aws.ToString(s.ClusterArn),
		TaskDefinitionARN: aws.ToString(s.TaskDefinition),
		DesiredCount:      int(s.DesiredCount),
		RunningCount:      int(s.RunningCount),
	}
}

// EnsureServiceOptions holds parameters for the EnsureService operation.
type EnsureServiceOptions struct {
	ClusterName         string
	ServiceName         string
	TaskDefinitionInput *RegisterTaskDefinitionInput
	DesiredCount        int
	SubnetIDs           []string
	SecurityGroupIDs    []string
	AssignPublicIP      bool
	TargetGroupARN      string
}

// EnsureService creates or updates an ECS Fargate service idempotently.
func EnsureService(ctx context.Context, client ECSClient, opts *EnsureServiceOptions) (*ECSService, error) {
	_, err := client.CreateCluster(ctx, opts.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("ensure service: %w", err)
	}

	td, err := client.RegisterTaskDefinition(ctx, opts.TaskDefinitionInput)
	if err != nil {
		return nil, fmt.Errorf("ensure service: %w", err)
	}

	services, err := client.DescribeServices(ctx, &DescribeServicesInput{
		Cluster:     opts.ClusterName,
		ServiceName: opts.ServiceName,
	})

	if errors.Is(err, ErrServiceNotFound) {
		svc, createErr := client.CreateService(ctx, &CreateECSServiceInput{
			Cluster:           opts.ClusterName,
			ServiceName:       opts.ServiceName,
			TaskDefinitionARN: td.TaskDefinitionARN,
			DesiredCount:      opts.DesiredCount,
			SubnetIDs:         opts.SubnetIDs,
			SecurityGroupIDs:  opts.SecurityGroupIDs,
			AssignPublicIP:    opts.AssignPublicIP,
			TargetGroupARN:    opts.TargetGroupARN,
		})
		if createErr != nil {
			return nil, fmt.Errorf("ensure service: %w", createErr)
		}
		return svc, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ensure service: describe: %w", err)
	}

	_ = services // service exists, update it
	svc, err := client.UpdateService(ctx, &UpdateECSServiceInput{
		Cluster:           opts.ClusterName,
		ServiceName:       opts.ServiceName,
		TaskDefinitionARN: td.TaskDefinitionARN,
		DesiredCount:      opts.DesiredCount,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure service: %w", err)
	}
	return svc, nil
}
