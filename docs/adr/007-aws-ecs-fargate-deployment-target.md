# ADR 007: AWS ECS Fargate as Deployment Target

## Status
Accepted

## Date
2026-03-12

## Context
Mint supports deploying generated MCP servers to GCP Cloud Run via `mint deploy gcp`. Users have requested AWS support with feature parity. The key GCP features that need AWS equivalents are: container image build and push, container service deployment, traffic splitting (canary), rollback, status reporting, secrets injection, IAM access control, health checks, and CI workflow generation.

AWS offers several container hosting options:
- **App Runner**: Simplest, closest to Cloud Run's developer experience, but lacks traffic splitting/canary deployments and weighted routing.
- **ECS Fargate**: Serverless containers with ALB-based weighted target groups for canary/traffic splitting. Full feature parity is achievable.
- **EKS Fargate**: Kubernetes-based, overly complex for single-service deployments.
- **Lambda with container images**: Cold start latency and 15-minute timeout make it unsuitable for long-running SSE connections.

## Decision
Use AWS ECS Fargate with Application Load Balancer (ALB) as the AWS deployment target. This provides:
- Serverless container execution (no EC2 management)
- ALB weighted target groups for canary traffic splitting
- Native integration with ECR, Secrets Manager, IAM, and CodeBuild
- Task definition revisions map naturally to Cloud Run revisions for rollback

The AWS deploy will follow the same interface-based adapter architecture established in the GCP deploy (see docs/adr/005-gcp-sdk-adapter-pattern.md). A new `internal/deploy/aws/` package will define its own orchestrator interfaces and SDK adapters.

## Consequences
**Positive:**
- Full feature parity with GCP deploy (canary, rollback, status, secrets, IAM, health checks).
- Users can deploy the same generated MCP server to either cloud with a flag change.
- ECS Fargate is well-supported by the AWS Go SDK v2.

**Negative:**
- ECS Fargate has more moving parts than Cloud Run (ALB, target groups, task definitions, ECS cluster, VPC). The adapter layer must manage this complexity.
- ALB requires a VPC; mint must either create one or require the user to specify an existing one.
- More AWS IAM roles required than the GCP equivalent.
