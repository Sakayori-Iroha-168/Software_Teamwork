/**
 * Admin API — Gateway OpenAPI admin-runtime-config, qa-settings,
 * qa-retrieval-tests, qa-metrics, knowledge, auth paths.
 *
 * All functions use doRequest from ./client.
 * Types imported from @/lib/types (camelCase, per OpenAPI).
 */

import type {
  CreateKnowledgeBaseRequest,
  CreateQAConfigVersionRequest,
  CreateQALLMConfigVersionRequest,
  KnowledgeBaseSummary,
  QAConfigVersion,
  QAIntentDistributionItem,
  QALLMConfigVersion,
  QALLMConnectionTest,
  QALLMConnectionTestRequest,
  QAMetricsOverview,
  QAMetricsTrend,
  QARetrievalTestRun,
  QARetrievalTestRunRequest,
  QATopQuery,
} from '@/lib/types'

import { doRequest } from './client'

// =========================================================================
// LLM Configuration
// =========================================================================

/** GET /llm-config-versions/current */
export function getCurrentLLMConfig(): Promise<QALLMConfigVersion> {
  return doRequest<QALLMConfigVersion>('/llm-config-versions/current')
}

/** POST /llm-config-versions */
export async function createLLMConfigVersion(
  config: CreateQALLMConfigVersionRequest,
): Promise<QALLMConfigVersion> {
  return doRequest<QALLMConfigVersion>('/llm-config-versions', {
    method: 'POST',
    body: JSON.stringify(config),
  })
}

/** POST /llm-connection-tests */
export async function testLLMConnection(
  params: QALLMConnectionTestRequest,
): Promise<QALLMConnectionTest> {
  return doRequest<QALLMConnectionTest>('/llm-connection-tests', {
    method: 'POST',
    body: JSON.stringify(params),
  })
}

// =========================================================================
// QA Configuration
// =========================================================================

/** GET /qa-config-versions/current */
export function getCurrentQAConfig(): Promise<QAConfigVersion> {
  return doRequest<QAConfigVersion>('/qa-config-versions/current')
}

/** POST /qa-config-versions */
export async function createQAConfigVersion(
  config: CreateQAConfigVersionRequest,
): Promise<QAConfigVersion> {
  return doRequest<QAConfigVersion>('/qa-config-versions', {
    method: 'POST',
    body: JSON.stringify(config),
  })
}

// =========================================================================
// Retrieval Test
// =========================================================================

/** POST /retrieval-test-runs */
export async function runRetrievalTest(
  params: QARetrievalTestRunRequest,
): Promise<QARetrievalTestRun> {
  return doRequest<QARetrievalTestRun>('/retrieval-test-runs', {
    method: 'POST',
    body: JSON.stringify(params),
  })
}

// =========================================================================
// QA Metrics
// =========================================================================

/** GET /qa-metrics/overview?days=N */
export function getQAMetricsOverview(days?: number): Promise<QAMetricsOverview> {
  const qs = days != null ? `?days=${days}` : ''
  return doRequest<QAMetricsOverview>(`/qa-metrics/overview${qs}`)
}

/** GET /qa-metrics/trend?days=N */
export function getQAMetricsTrend(days?: number): Promise<QAMetricsTrend> {
  const qs = days != null ? `?days=${days}` : '?days=30'
  return doRequest<QAMetricsTrend>(`/qa-metrics/trend${qs}`)
}

/** GET /qa-metrics/top-queries?limit=N&days=N */
export async function getQATopQueries(limit?: number, days?: number): Promise<QATopQuery[]> {
  const params = new URLSearchParams()
  if (limit != null) params.set('limit', String(limit))
  if (days != null) params.set('days', String(days))
  const qs = params.toString()
  return doRequest<QATopQuery[]>(`/qa-metrics/top-queries${qs ? `?${qs}` : ''}`)
}

/** GET /qa-metrics/intent-distribution?days=N */
export async function getQAIntentDistribution(days?: number): Promise<QAIntentDistributionItem[]> {
  const qs = days != null ? `?days=${days}` : ''
  return doRequest<QAIntentDistributionItem[]>(`/qa-metrics/intent-distribution${qs}`)
}

// =========================================================================
// Knowledge Bases
// =========================================================================

/** GET /knowledge-bases */
export function listKnowledgeBases(): Promise<KnowledgeBaseSummary[]> {
  return doRequest<KnowledgeBaseSummary[]>('/knowledge-bases')
}

/** POST /knowledge-bases */
export async function createKnowledgeBase(
  params: CreateKnowledgeBaseRequest,
): Promise<KnowledgeBaseSummary> {
  return doRequest<KnowledgeBaseSummary>('/knowledge-bases', {
    method: 'POST',
    body: JSON.stringify(params),
  })
}

/** DELETE /knowledge-bases/{id} */
export async function deleteKnowledgeBase(id: string): Promise<void> {
  await doRequest<void>(`/knowledge-bases/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

// =========================================================================
// User Management (stubs — reserved)
// =========================================================================

/** GET /users */
export function listUsers(): Promise<Record<string, unknown>[]> {
  return doRequest<Record<string, unknown>[]>('/users')
}

/** POST /users */
export async function createUser(body: Record<string, unknown>): Promise<Record<string, unknown>> {
  return doRequest<Record<string, unknown>>('/users', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

/** PUT /users/{id} */
export async function updateUser(
  id: string,
  body: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  return doRequest<Record<string, unknown>>(`/users/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

/** DELETE /users/{id} */
export async function deleteUser(id: string): Promise<void> {
  await doRequest<void>(`/users/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

// =========================================================================
// Role Management (stubs — reserved)
// =========================================================================

/** GET /roles */
export function listRoles(): Promise<Record<string, unknown>[]> {
  return doRequest<Record<string, unknown>[]>('/roles')
}

/** POST /roles */
export async function createRole(body: Record<string, unknown>): Promise<Record<string, unknown>> {
  return doRequest<Record<string, unknown>>('/roles', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

/** PUT /roles/{id} */
export async function updateRole(
  id: string,
  body: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  return doRequest<Record<string, unknown>>(`/roles/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

/** DELETE /roles/{id} */
export async function deleteRole(id: string): Promise<void> {
  await doRequest<void>(`/roles/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
