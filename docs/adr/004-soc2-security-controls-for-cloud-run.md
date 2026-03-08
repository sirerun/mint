# ADR 004: SOC2 Security Controls for Cloud Run Deployments

## Status
Accepted

## Date
2026-03-07

## Context
Deployed MCP servers handle API credentials and potentially sensitive data. SOC2 Type II compliance requires controls across five trust service criteria: security, availability, processing integrity, confidentiality, and privacy. Cloud Run provides platform-level compliance, but application-level controls must be configured correctly.

## Decision
Enforce the following security controls by default in all `mint deploy gcp` deployments:

**Authentication and Authorization:**
- Cloud Run services deployed with `--no-allow-unauthenticated` by default. IAM-based access control.
- Service account per deployment with least-privilege permissions (only the APIs the MCP server needs).
- `--allow-unauthenticated` requires explicit `--public` flag with confirmation.

**Encryption:**
- All traffic encrypted in transit via Cloud Run's built-in TLS (minimum TLS 1.2).
- Secrets stored in Secret Manager, mounted as environment variables at runtime. Never baked into container images.
- Container images stored in Artifact Registry with encryption at rest (Google-managed keys by default, CMEK optional via `--kms-key` flag).

**Network Security:**
- VPC connector configured when `--vpc` flag is provided, enabling private networking.
- Ingress restricted to internal-only when `--internal` flag is set.
- Cloud Armor WAF policy attached when `--waf` flag is provided.

**Audit and Monitoring:**
- Cloud Audit Logs enabled by default for all Cloud Run admin operations.
- Cloud Logging for application logs (stdout/stderr captured automatically by Cloud Run).
- Deployment metadata (commit SHA, spec hash, deployer identity) recorded as Cloud Run service labels.

**Container Security:**
- Multi-stage Docker build: build stage uses Go builder, runtime stage uses distroless base image (`gcr.io/distroless/static-debian12`).
- No shell, no package manager in runtime container.
- Container runs as non-root user.
- Binary Authorization optionally enforced via `--binary-auth` flag.

## Consequences
**Positive:**
- Deployments are secure by default. Users must explicitly opt in to less secure configurations.
- Audit trail is automatic. No additional tooling needed for compliance evidence.
- Distroless base image eliminates entire classes of container vulnerabilities.

**Negative:**
- IAM-authenticated services require clients to present identity tokens, which adds complexity for non-GCP callers. Mitigated by documenting how to generate tokens.
- Secret Manager adds a dependency and per-access cost (negligible for typical usage).
- Distroless containers cannot be debugged with shell access. Debug builds can use `--debug-image` flag to use a standard base image in non-production environments.
