#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

docker compose -f docker-compose.db.yml up -d --build postgres
docker compose -f docker-compose.db.yml up --build migrate

port="${QA_POSTGRES_PORT:-5433}"
echo
echo "QA PostgreSQL is ready on localhost:${port}"
echo "Connection string: postgres://qa_app:qa_app_dev@localhost:${port}/qa_system?sslmode=disable"
