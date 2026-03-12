package gcp

import "testing"

func TestIAMPolicyAdapterInterfaceCheck(t *testing.T) {
	var _ IAMPolicyClient = (*IAMPolicyAdapter)(nil)
}

func TestIAMServiceAccountAdapterInterfaceCheck(t *testing.T) {
	var _ IAMClient = (*IAMServiceAccountAdapter)(nil)
}

func TestIAMPolicyConversionRoundTrip(t *testing.T) {
	original := &IAMPolicy{
		Bindings: []IAMBinding{
			{Role: "roles/run.invoker", Members: []string{"allUsers"}},
			{Role: "roles/editor", Members: []string{"user:a@b.com", "user:c@d.com"}},
		},
	}

	pb := iamPolicyToPb(original)
	result := iamPolicyFromPb(pb)

	if len(result.Bindings) != len(original.Bindings) {
		t.Fatalf("bindings count: got %d, want %d", len(result.Bindings), len(original.Bindings))
	}
	for i, b := range result.Bindings {
		if b.Role != original.Bindings[i].Role {
			t.Errorf("binding %d role: got %q, want %q", i, b.Role, original.Bindings[i].Role)
		}
		if len(b.Members) != len(original.Bindings[i].Members) {
			t.Errorf("binding %d members count: got %d, want %d", i, len(b.Members), len(original.Bindings[i].Members))
		}
		for j, m := range b.Members {
			if m != original.Bindings[i].Members[j] {
				t.Errorf("binding %d member %d: got %q, want %q", i, j, m, original.Bindings[i].Members[j])
			}
		}
	}
}
