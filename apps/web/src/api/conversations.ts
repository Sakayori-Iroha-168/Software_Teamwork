/**
 * QA Sessions CRUD — Gateway OpenAPI qa-sessions paths.
 *
 * All functions use doRequest / listRequest from ./client.
 * Types imported from @/lib/types (camelCase, per OpenAPI).
 */

import type { QAMessage, QASession } from '@/lib/types'

import { doRequest, listRequest, type ListResponse } from './client'

// ---------------------------------------------------------------------------
// POST /qa-sessions
// ---------------------------------------------------------------------------

export async function createSession(title?: string): Promise<QASession> {
  return doRequest<QASession>('/qa-sessions', {
    method: 'POST',
    body: JSON.stringify({ title }),
  })
}

// ---------------------------------------------------------------------------
// GET /qa-sessions?page=&pageSize=&sort=-updatedAt
// ---------------------------------------------------------------------------

export async function listSessions(page = 1, pageSize = 20): Promise<ListResponse<QASession>> {
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
    sort: '-updatedAt',
  })
  return listRequest<QASession>(`/qa-sessions?${params}`)
}

// ---------------------------------------------------------------------------
// GET /qa-sessions/{sessionId}
// ---------------------------------------------------------------------------

export async function getSession(sessionId: string): Promise<QASession> {
  return doRequest<QASession>(`/qa-sessions/${encodeURIComponent(sessionId)}`)
}

// ---------------------------------------------------------------------------
// PATCH /qa-sessions/{sessionId}
// ---------------------------------------------------------------------------

export async function renameSession(sessionId: string, title: string): Promise<QASession> {
  return doRequest<QASession>(`/qa-sessions/${encodeURIComponent(sessionId)}`, {
    method: 'PATCH',
    body: JSON.stringify({ title }),
  })
}

// ---------------------------------------------------------------------------
// DELETE /qa-sessions/{sessionId}
// ---------------------------------------------------------------------------

export async function deleteSession(sessionId: string): Promise<void> {
  await doRequest<void>(`/qa-sessions/${encodeURIComponent(sessionId)}`, { method: 'DELETE' })
}

// ---------------------------------------------------------------------------
// GET /qa-sessions/{sessionId}/messages
// ---------------------------------------------------------------------------

export async function getSessionMessages(sessionId: string): Promise<QAMessage[]> {
  return doRequest<QAMessage[]>(`/qa-sessions/${encodeURIComponent(sessionId)}/messages`)
}
