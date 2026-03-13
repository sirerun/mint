package aws

import (
	"context"
	"fmt"
	"strings"
)

var requiredServices = []string{
	"ecs",
	"ecr",
	"elasticloadbalancing",
	"codebuild",
	"secretsmanager",
	"iam",
	"sts",
}

// ServiceChecker verifies that an AWS service is accessible.
type ServiceChecker interface {
	CheckService(ctx context.Context, service string) error
}

// CheckRequiredServices verifies that all required AWS services are accessible
// in the target region. If any are inaccessible, it returns an error listing
// them along with guidance on IAM permissions and region support.
func CheckRequiredServices(ctx context.Context, checker ServiceChecker, region string) error {
	var unavailable []string
	for _, svc := range requiredServices {
		if err := checker.CheckService(ctx, svc); err != nil {
			unavailable = append(unavailable, svc)
		}
	}
	if len(unavailable) > 0 {
		return fmt.Errorf("required AWS services not accessible in region %s:\n  %s\n\nEnsure your IAM permissions include access to these services and the region supports them",
			region, strings.Join(unavailable, "\n  "))
	}
	return nil
}
