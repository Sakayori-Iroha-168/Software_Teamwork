# S-05 Local Integration Environment

## Goal

Provide a local/demo integration baseline for issue #122 so new contributors can start the core backend dependency loop with Docker Compose or an equivalent command path. The baseline should connect shared infrastructure and optional services without requiring the frontend to call internal services directly.

## What I Already Know

* Task number: `S-05`.
* Primary team: `Special`.
* Priority: `P1`.
* Batch: `Batch 4`.
* Module: `deploy`.
* Dependency tasks: `#73`, `#76`, `#79`, `#81`, `#87`, `#97`, `#119`.
* Suggested branch: `Special/chore/local-integration-env`.
* Issue: `#122`.
* Authoritative references:
  * `docs/architecture/technology-decisions.md`
  * `docs/collaboration/repository-settings.md`
  * `docs/services/knowledge/docs/implementation.md`
  * `docs/services/document/README.md`

## Requirements

* Provide a local Docker Compose or equivalent startup path connecting:
  * `postgres`
  * `redis`
  * `qdrant`
  * `minio`
  * `gateway`
  * `auth`
  * `file`
  * `knowledge`
  * `qa`
  * `document`
  * `ai-gateway`
* Support optional services where the implementation baseline does not yet provide runnable service binaries/images.
* Maintain environment examples, service port documentation, health checks, `readyz` dependency behavior, and initialization scripts.
* Provide minimal seed data:
  * administrator user
  * model profile placeholder
  * report types
  * example knowledge base
* Document request-id troubleshooting flow and common dependency failures.
* Keep demo secrets non-real and clearly marked as examples.

## Acceptance Criteria

* [ ] A new member can follow documentation to start the core local integration dependencies.
* [ ] Each service has clear environment variable documentation.
* [ ] Demo environment initialization steps are reproducible.
* [ ] Health checks can identify dependencies that are not ready.
* [ ] Example secrets are not real secrets.
* [ ] Startup documentation does not require frontend direct access to internal services.

## Definition of Done

* Relevant dependency code and docs for `#73`, `#76`, `#79`, `#81`, `#87`, `#97`, and `#119` have been reviewed.
* Applicable `.trellis/spec/` coding and documentation guidelines have been read and followed.
* Lint/typecheck or the most relevant repository verification commands have been run.
* Documentation and examples are updated with reproducible local/demo commands.
* PR guidance is available: target `develop`, Conventional Commits, scope/verification/risks/issue in PR body.

## Out of Scope

* Production deployment hardening.
* Real production secrets, credentials, or cloud infrastructure setup.
* Forcing all optional services to be implemented if their dependency tasks have not produced runnable code yet.

## Technical Notes

* This task is local/demo integration only.
* Project sync status from task brief: `blocked`.
* Initial implementation should prefer the repository's existing service layout, environment names, and health endpoint conventions over inventing new service contracts.
