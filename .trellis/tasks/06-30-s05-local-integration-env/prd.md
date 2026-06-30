# S-05 Local Integration Environment

## Goal

Provide a reproducible local/demo integration baseline for backend services so new contributors can start core dependencies and service loops without connecting the frontend directly to internal services.

## Requirements

- Provide Docker Compose or equivalent local startup wiring for PostgreSQL, Redis, Qdrant, MinIO, gateway, auth, file, knowledge, qa, document, and optional ai-gateway.
- Maintain environment examples, service ports, health checks, readiness dependencies, and initialization scripts.
- Provide minimal seed data for an admin user, model profile placeholders, report types, and an example knowledge base.
- Document request id troubleshooting and common dependency failures.
- Keep sample secrets clearly non-production placeholders.

## Acceptance Criteria

- [ ] Health checks and readyz endpoints can identify unready dependencies.
- [ ] Example secrets are not real keys.
- [ ] Startup documentation routes browser/frontend traffic through gateway rather than direct internal services.
- [ ] New contributors can follow documented steps to start the core closed-loop dependencies.
- [ ] Docker image prerequisites are documented because the current Docker environment lacks required images.

## Definition of Done

- Deployment files and docs are updated.
- Existing service contracts and repository docs are respected.
- Relevant non-Docker verification is run where possible.
- Docker verification gap and required image installation commands are reported.

## Technical Approach

Use a repo-root deploy baseline with Docker Compose, service-specific environment files, database initialization SQL, and documentation. Prefer existing service Dockerfiles and migrations over new service code. Treat ai-gateway as optional in the local/demo loop while documenting how to enable it.

## Out of Scope

- Production deployment hardening.
- Real provider credentials or production-grade secret management.
- Frontend changes unless needed to preserve documented routing assumptions.

## Technical Notes

- Authoritative docs from the issue: `docs/architecture/technology-decisions.md`, `docs/collaboration/repository-settings.md`, `docs/services/knowledge/docs/implementation.md`, `docs/services/document/README.md`.
- Relevant services live under `services/`.
- Infrastructure dependencies are PostgreSQL, Redis, Qdrant, and MinIO.
