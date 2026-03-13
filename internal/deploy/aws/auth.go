package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Credentials wraps resolved AWS configuration.
type Credentials struct {
	Config    aws.Config
	Region    string
	AccountID string
}

// STSClient is the subset of the STS API used for authentication.
type STSClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// Authenticate resolves AWS credentials via the SDK v2 default chain.
// If stsClient is nil, a real STS client is created from the loaded config.
func Authenticate(ctx context.Context, region string, stsClient STSClient) (*Credentials, error) {
	if region == "" {
		return nil, fmt.Errorf("AWS region is required. Use --region flag or set AWS_REGION")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS credentials not found. Configure credentials via environment variables, ~/.aws/credentials, or IAM role: %w", err)
	}

	if stsClient == nil {
		stsClient = sts.NewFromConfig(cfg)
	}

	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to verify AWS credentials. Ensure valid credentials are configured: %w", err)
	}

	accountID := ""
	if identity.Account != nil {
		accountID = *identity.Account
	}
	if accountID == "" {
		return nil, fmt.Errorf("AWS account ID could not be resolved from credentials")
	}

	return &Credentials{
		Config:    cfg,
		Region:    region,
		AccountID: accountID,
	}, nil
}
