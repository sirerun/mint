package azure

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// fakeTokenCredential implements azcore.TokenCredential for testing.
type fakeTokenCredential struct{}

func (f *fakeTokenCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{}, nil
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID string
		resourceGroup  string
		credErr        error
		wantErr        string
	}{
		{
			name:           "missing subscription ID returns error",
			subscriptionID: "",
			resourceGroup:  "my-rg",
			wantErr:        "AZURE_SUBSCRIPTION_ID is required",
		},
		{
			name:           "missing resource group returns error",
			subscriptionID: "sub-123",
			resourceGroup:  "",
			wantErr:        "AZURE_RESOURCE_GROUP is required",
		},
		{
			name:           "credential error returns helpful message",
			subscriptionID: "sub-123",
			resourceGroup:  "my-rg",
			credErr:        fmt.Errorf("no credentials available"),
			wantErr:        "azure credentials not found",
		},
		{
			name:           "successful authentication",
			subscriptionID: "sub-456",
			resourceGroup:  "prod-rg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origSub := getSubscriptionID
			origRG := getResourceGroup
			origCred := newDefaultCredential
			t.Cleanup(func() {
				getSubscriptionID = origSub
				getResourceGroup = origRG
				newDefaultCredential = origCred
			})

			getSubscriptionID = func() string { return tt.subscriptionID }
			getResourceGroup = func() string { return tt.resourceGroup }
			newDefaultCredential = func(options *azidentity.DefaultAzureCredentialOptions) (azcore.TokenCredential, error) {
				if tt.credErr != nil {
					return nil, tt.credErr
				}
				return &fakeTokenCredential{}, nil
			}

			var stderr bytes.Buffer
			creds, err := Authenticate(context.Background(), &stderr)

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
			if creds.SubscriptionID != tt.subscriptionID {
				t.Errorf("SubscriptionID = %q, want %q", creds.SubscriptionID, tt.subscriptionID)
			}
			if creds.ResourceGroup != tt.resourceGroup {
				t.Errorf("ResourceGroup = %q, want %q", creds.ResourceGroup, tt.resourceGroup)
			}
			if creds.Credential == nil {
				t.Error("expected non-nil Credential")
			}
			if !strings.Contains(stderr.String(), "Authenticated with Azure") {
				t.Errorf("expected stderr output, got %q", stderr.String())
			}
		})
	}
}
