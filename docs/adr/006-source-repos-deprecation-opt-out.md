# ADR 006: Default Source Repos to Opt-Out Due to API Deprecation

## Status
Accepted

## Date
2026-03-11

## Context
The `mint deploy gcp` command includes a Source Repository integration that mirrors source code to Google Cloud Source Repositories for auditability. However, the Cloud Source Repositories API is being deprecated by Google. Continuing to enable it by default would cause failures for new GCP projects where the API is not available.

## Decision
Default the `--no-source-repo` flag to true, making the Source Repository push opt-in rather than opt-out. When the Source Repository adapter is used, log a deprecation warning. The adapter implementation uses the REST API (`google.golang.org/api/sourcerepo/v1`) since no gRPC client library exists for this service.

## Consequences
- Positive: New users are not affected by the deprecated API. Deployments work without enabling Cloud Source Repositories.
- Positive: Existing users who rely on the feature can still opt in explicitly.
- Negative: Users who previously relied on automatic source mirroring must now pass an explicit flag to enable it.
- Negative: The adapter code is maintained for a deprecated service. It should be removed entirely once Google fully shuts down the API.
