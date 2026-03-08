# ADR 002: GCP Go SDK Over Terraform for Infrastructure Provisioning

## Status
Accepted

## Date
2026-03-07

## Context
The `mint deploy` command needs to provision GCP resources (Cloud Run services, Artifact Registry repositories, IAM policies, Secret Manager secrets). Two approaches were considered:

1. **Terraform**: Declare infrastructure as HCL files, shell out to `terraform apply`. Requires Terraform binary installed. State management adds complexity. Well-suited for long-lived infrastructure but heavyweight for deploying a single service.

2. **GCP Go SDK**: Use `cloud.google.com/go` libraries to call GCP APIs directly from the mint binary. No external tool dependencies. Programmatic control over deployment flow. State is the GCP project itself (check-before-create pattern).

## Decision
Use the GCP Go SDK (`cloud.google.com/go`) for all infrastructure provisioning. This keeps mint as a single binary with no external tool dependencies. The SDK provides typed, compile-time-safe access to all required GCP APIs.

Required SDK packages:
- `cloud.google.com/go/run/apiv2` -- Cloud Run service management
- `cloud.google.com/go/artifactregistry/apiv1` -- Container image registry
- `cloud.google.com/go/cloudbuild/apiv1/v2` -- Container builds
- `cloud.google.com/go/secretmanager/apiv1` -- Secret storage
- `cloud.google.com/go/iam` -- IAM policy management
- `cloud.google.com/go/sourcerepo/apiv1` -- Cloud Source Repositories

## Consequences
**Positive:**
- No external dependencies (Terraform, gcloud CLI) required at runtime.
- Single binary philosophy preserved.
- Full programmatic control over deployment ordering and error handling.
- Idempotent operations via check-before-create pattern.
- AI-native release pipeline benefits from direct API calls (faster, no HCL templating).

**Negative:**
- More Go code to write and maintain compared to Terraform HCL.
- GCP SDK API changes require code updates (mitigated by pinning SDK versions).
- Users who prefer Terraform cannot reuse mint's deployment logic as HCL modules.
- No built-in state tracking -- must query GCP APIs to determine current state.
