package aws

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type mockSTSClient struct {
	output *sts.GetCallerIdentityOutput
	err    error
}

func (m *mockSTSClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return m.output, m.err
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		client     STSClient
		wantErr    string
		wantAcct   string
		wantRegion string
	}{
		{
			name:    "missing region returns error",
			region:  "",
			client:  &mockSTSClient{},
			wantErr: "AWS region is required",
		},
		{
			name:   "STS failure returns helpful error",
			region: "us-east-1",
			client: &mockSTSClient{
				err: fmt.Errorf("expired credentials"),
			},
			wantErr: "failed to verify AWS credentials",
		},
		{
			name:   "empty account ID returns error",
			region: "us-west-2",
			client: &mockSTSClient{
				output: &sts.GetCallerIdentityOutput{
					Account: aws.String(""),
				},
			},
			wantErr: "AWS account ID could not be resolved",
		},
		{
			name:   "successful authentication",
			region: "eu-west-1",
			client: &mockSTSClient{
				output: &sts.GetCallerIdentityOutput{
					Account: aws.String("123456789012"),
					Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
				},
			},
			wantAcct:   "123456789012",
			wantRegion: "eu-west-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := Authenticate(context.Background(), tt.region, tt.client)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if creds.AccountID != tt.wantAcct {
				t.Errorf("AccountID = %q, want %q", creds.AccountID, tt.wantAcct)
			}
			if creds.Region != tt.wantRegion {
				t.Errorf("Region = %q, want %q", creds.Region, tt.wantRegion)
			}
		})
	}
}
