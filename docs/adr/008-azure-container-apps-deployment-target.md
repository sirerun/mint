# ADR 008: Azure Container Apps as Deployment Target

## Status
Accepted

## Date
2026-03-13

## Context
Mint supports deploying generated MCP servers to GCP Cloud Run and AWS ECS Fargate. Enterprise customers on Azure have no equivalent. Azure offers several container hosting options:

- **Azure Container Apps (ACA)**: Serverless containers built on Kubernetes (KEDA + Envoy). Closest to Cloud Run in developer experience. Supports traffic splitting natively via revision-based routing. Managed HTTPS with custom domains. Built-in Dapr integration (optional). Consumption-based pricing.
- **Azure Container Instances (ACI)**: Single container instances, no traffic splitting or revision management.
- **Azure Kubernetes Service (AKS)**: Full Kubernetes, overly complex for single-service MCP server deployments.
- **Azure App Service**: Supports containers but traffic splitting is limited to deployment slots (max 5), and the model is VM-based.

## Decision
Use Azure Container Apps as the Azure deployment target. ACA provides:
- Serverless container execution (no VM or cluster management)
- Native revision-based traffic splitting for canary deployments (identical model to Cloud Run)
- Built-in container registry integration via Azure Container Registry (ACR)
- Managed identity for keyless auth from GitHub Actions via OIDC federation
- Secrets injection from Azure Key Vault
- Auto-scaling with KEDA scale rules

The Azure deploy follows the same three-layer adapter architecture (Deployer -> Bridge -> SDK) established by GCP and AWS. A new `internal/deploy/azure/` package will use the Azure SDK for Go (`github.com/Azure/azure-sdk-for-go/sdk`).

## Consequences
**Positive:**
- Full feature parity with GCP and AWS deploy (canary, rollback, status, secrets, IAM, health checks).
- ACA's revision-based traffic splitting maps directly to Cloud Run's model, simplifying the adapter layer compared to AWS's ALB-based approach.
- Azure SDK for Go is well-maintained and follows similar patterns to AWS SDK v2.

**Negative:**
- Adds a third cloud SDK dependency tree to go.mod.
- ACA requires a Container Apps Environment (similar to an ECS cluster) which must be created or referenced.
- Azure RBAC model differs from both GCP IAM and AWS IAM; the IAM adapter needs Azure-specific role assignments.
