$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

docker compose -f docker-compose.db.yml up -d --build postgres
docker compose -f docker-compose.db.yml up --build migrate

Write-Host ""
Write-Host "QA PostgreSQL is ready on localhost:${env:QA_POSTGRES_PORT:-5433}"
Write-Host "Connection string: postgres://qa_app:qa_app_dev@localhost:${env:QA_POSTGRES_PORT:-5433}/qa_system?sslmode=disable"
