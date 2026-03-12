package gcp

import (
	"context"
	"fmt"

	run "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// cloudRunClients holds the shared GCP SDK clients for Cloud Run.
type cloudRunClients struct {
	services  *run.ServicesClient
	revisions *run.RevisionsClient
}

// CloudRunAdapter groups the four adapter structs that implement the Cloud Run
// interfaces. Because Go does not allow two methods with the same name but
// different signatures on one struct, we split them into separate types.
type CloudRunAdapter struct {
	Service  *CloudRunServiceAdapter
	Status   *CloudRunStatusAdapter
	Revision *CloudRunRevisionAdapter
	Traffic  *CloudRunTrafficAdapter
}

// NewCloudRunAdapter creates SDK clients and returns an adapter containing all
// four sub-adapters. The caller should call Close when done.
func NewCloudRunAdapter(ctx context.Context) (*CloudRunAdapter, error) {
	svc, err := run.NewServicesClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating services client: %w", err)
	}
	rev, err := run.NewRevisionsClient(ctx)
	if err != nil {
		svc.Close()
		return nil, fmt.Errorf("creating revisions client: %w", err)
	}
	clients := &cloudRunClients{services: svc, revisions: rev}
	return &CloudRunAdapter{
		Service:  &CloudRunServiceAdapter{clients: clients},
		Status:   &CloudRunStatusAdapter{clients: clients},
		Revision: &CloudRunRevisionAdapter{clients: clients},
		Traffic:  &CloudRunTrafficAdapter{clients: clients},
	}, nil
}

// Close releases the underlying gRPC connections.
func (a *CloudRunAdapter) Close() error {
	errS := a.Service.clients.services.Close()
	errR := a.Revision.clients.revisions.Close()
	if errS != nil {
		return errS
	}
	return errR
}

// ---------------------------------------------------------------------------
// CloudRunServiceAdapter implements CloudRunClient.
// ---------------------------------------------------------------------------

// CloudRunServiceAdapter implements the CloudRunClient interface.
type CloudRunServiceAdapter struct {
	clients *cloudRunClients
}

var _ CloudRunClient = (*CloudRunServiceAdapter)(nil)

// ServicesClient returns the underlying run.ServicesClient for sharing with
// other adapters that need it (e.g., IAMPolicyAdapter).
func (a *CloudRunServiceAdapter) ServicesClient() *run.ServicesClient {
	return a.clients.services
}

// GetService retrieves a Cloud Run service by its full resource name.
func (a *CloudRunServiceAdapter) GetService(ctx context.Context, name string) (*Service, error) {
	svc, err := a.clients.services.GetService(ctx, &runpb.GetServiceRequest{Name: name})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return serviceFromPb(svc), nil
}

// CreateService creates a new Cloud Run service and waits for the LRO.
func (a *CloudRunServiceAdapter) CreateService(ctx context.Context, config *ServiceConfig) (*Service, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", config.ProjectID, config.Region)
	pbSvc := serviceConfigToPb(config)

	op, err := a.clients.services.CreateService(ctx, &runpb.CreateServiceRequest{
		Parent:    parent,
		Service:   pbSvc,
		ServiceId: config.ServiceName,
	})
	if err != nil {
		return nil, err
	}
	result, err := op.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return serviceFromPb(result), nil
}

// UpdateService updates an existing Cloud Run service and waits for the LRO.
func (a *CloudRunServiceAdapter) UpdateService(ctx context.Context, config *ServiceConfig) (*Service, error) {
	pbSvc := serviceConfigToPb(config)
	pbSvc.Name = ServiceFullName(config.ProjectID, config.Region, config.ServiceName)

	op, err := a.clients.services.UpdateService(ctx, &runpb.UpdateServiceRequest{
		Service: pbSvc,
	})
	if err != nil {
		return nil, err
	}
	result, err := op.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return serviceFromPb(result), nil
}

// ---------------------------------------------------------------------------
// CloudRunStatusAdapter implements StatusClient.
// ---------------------------------------------------------------------------

// CloudRunStatusAdapter implements the StatusClient interface.
type CloudRunStatusAdapter struct {
	clients *cloudRunClients
}

var _ StatusClient = (*CloudRunStatusAdapter)(nil)

// GetService retrieves service status by its full resource name.
func (a *CloudRunStatusAdapter) GetService(ctx context.Context, name string) (*ServiceStatus, error) {
	svc, err := a.clients.services.GetService(ctx, &runpb.GetServiceRequest{Name: name})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return serviceStatusFromPb(svc), nil
}

// ListRevisions lists revisions for a service, returning status information.
func (a *CloudRunStatusAdapter) ListRevisions(ctx context.Context, serviceName string) ([]RevisionStatus, error) {
	it := a.clients.revisions.ListRevisions(ctx, &runpb.ListRevisionsRequest{Parent: serviceName})
	var revisions []RevisionStatus
	for {
		rev, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, revisionStatusFromPb(rev))
	}
	return revisions, nil
}

// ---------------------------------------------------------------------------
// CloudRunRevisionAdapter implements RevisionClient.
// ---------------------------------------------------------------------------

// CloudRunRevisionAdapter implements the RevisionClient interface.
type CloudRunRevisionAdapter struct {
	clients *cloudRunClients
}

var _ RevisionClient = (*CloudRunRevisionAdapter)(nil)

// ListRevisions lists revisions for a service.
func (a *CloudRunRevisionAdapter) ListRevisions(ctx context.Context, serviceName string) ([]Revision, error) {
	it := a.clients.revisions.ListRevisions(ctx, &runpb.ListRevisionsRequest{Parent: serviceName})
	var revisions []Revision
	for {
		rev, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, revisionFromPb(rev))
	}
	return revisions, nil
}

// UpdateTraffic routes the given percentage of traffic to the specified
// revision by updating the service's traffic configuration.
func (a *CloudRunRevisionAdapter) UpdateTraffic(ctx context.Context, serviceName string, revisionName string, percent int) error {
	svc, err := a.clients.services.GetService(ctx, &runpb.GetServiceRequest{Name: serviceName})
	if err != nil {
		return err
	}
	svc.Traffic = []*runpb.TrafficTarget{
		{
			Type:     runpb.TrafficTargetAllocationType_TRAFFIC_TARGET_ALLOCATION_TYPE_REVISION,
			Revision: revisionName,
			Percent:  int32(percent),
		},
	}
	op, err := a.clients.services.UpdateService(ctx, &runpb.UpdateServiceRequest{Service: svc})
	if err != nil {
		return err
	}
	_, err = op.Wait(ctx)
	return err
}

// ---------------------------------------------------------------------------
// CloudRunTrafficAdapter implements TrafficClient.
// ---------------------------------------------------------------------------

// CloudRunTrafficAdapter implements the TrafficClient interface.
type CloudRunTrafficAdapter struct {
	clients *cloudRunClients
}

var _ TrafficClient = (*CloudRunTrafficAdapter)(nil)

// GetTraffic returns the current traffic targets for a service.
func (a *CloudRunTrafficAdapter) GetTraffic(ctx context.Context, serviceName string) ([]TrafficTarget, error) {
	svc, err := a.clients.services.GetService(ctx, &runpb.GetServiceRequest{Name: serviceName})
	if err != nil {
		return nil, err
	}
	targets := make([]TrafficTarget, len(svc.Traffic))
	for i, t := range svc.Traffic {
		targets[i] = TrafficTarget{
			RevisionName: t.Revision,
			Percent:      int(t.Percent),
			Tag:          t.Tag,
		}
	}
	return targets, nil
}

// SetTraffic replaces the traffic configuration for a service.
func (a *CloudRunTrafficAdapter) SetTraffic(ctx context.Context, serviceName string, targets []TrafficTarget) error {
	svc, err := a.clients.services.GetService(ctx, &runpb.GetServiceRequest{Name: serviceName})
	if err != nil {
		return err
	}
	pbTargets := make([]*runpb.TrafficTarget, len(targets))
	for i, t := range targets {
		pbTargets[i] = &runpb.TrafficTarget{
			Type:     runpb.TrafficTargetAllocationType_TRAFFIC_TARGET_ALLOCATION_TYPE_REVISION,
			Revision: t.RevisionName,
			Percent:  int32(t.Percent),
			Tag:      t.Tag,
		}
	}
	svc.Traffic = pbTargets
	op, err := a.clients.services.UpdateService(ctx, &runpb.UpdateServiceRequest{Service: svc})
	if err != nil {
		return err
	}
	_, err = op.Wait(ctx)
	return err
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

func serviceConfigToPb(config *ServiceConfig) *runpb.Service {
	port := int32(config.Port)
	if port == 0 {
		port = 8080
	}

	var envVars []*runpb.EnvVar
	for k, v := range config.EnvVars {
		envVars = append(envVars, &runpb.EnvVar{
			Name:   k,
			Values: &runpb.EnvVar_Value{Value: v},
		})
	}

	return &runpb.Service{
		Template: &runpb.RevisionTemplate{
			Containers: []*runpb.Container{{
				Image: config.ImageURI,
				Ports: []*runpb.ContainerPort{{ContainerPort: port}},
				Env:   envVars,
			}},
			Scaling: &runpb.RevisionScaling{
				MinInstanceCount: int32(config.MinInstances),
				MaxInstanceCount: int32(config.MaxInstances),
			},
		},
		Labels: config.Labels,
	}
}

func serviceFromPb(svc *runpb.Service) *Service {
	return &Service{
		Name:         svc.Name,
		URL:          svc.Uri,
		RevisionName: svc.LatestCreatedRevision,
		Status:       conditionStatus(svc.Conditions),
	}
}

func serviceStatusFromPb(svc *runpb.Service) *ServiceStatus {
	ss := &ServiceStatus{
		Name:   svc.Name,
		URL:    svc.Uri,
		Labels: svc.Labels,
	}
	if svc.CreateTime != nil {
		ss.CreateTime = svc.CreateTime.AsTime()
	}
	if svc.UpdateTime != nil {
		ss.UpdateTime = svc.UpdateTime.AsTime()
	}
	return ss
}

func revisionFromPb(rev *runpb.Revision) Revision {
	r := Revision{
		Name:   rev.Name,
		Active: isRevisionActive(rev.Conditions),
	}
	if rev.CreateTime != nil {
		r.CreateTime = rev.CreateTime.AsTime()
	}
	return r
}

func revisionStatusFromPb(rev *runpb.Revision) RevisionStatus {
	rs := RevisionStatus{
		Name:   rev.Name,
		Active: isRevisionActive(rev.Conditions),
	}
	if rev.CreateTime != nil {
		rs.CreateTime = rev.CreateTime.AsTime()
	}
	return rs
}

func conditionStatus(conditions []*runpb.Condition) string {
	for _, c := range conditions {
		if c.Type == "Ready" {
			switch c.State {
			case runpb.Condition_CONDITION_SUCCEEDED:
				return "Ready"
			case runpb.Condition_CONDITION_FAILED:
				return "Failed"
			default:
				return "Pending"
			}
		}
	}
	return "Unknown"
}

func isRevisionActive(conditions []*runpb.Condition) bool {
	for _, c := range conditions {
		if c.Type == "Active" && c.State == runpb.Condition_CONDITION_SUCCEEDED {
			return true
		}
	}
	return false
}
