/**
 * API client aligned with the Gateway OpenAPI specification.
 *
 * Envelope contracts:
 * - Success  : { data: T, requestId: string }
 * - List     : { data: T[], page: { page, pageSize, total }, requestId: string }
 * - Error    : { error: { code: string, message: string, requestId: string, fields?: Record<string,string> } }
 *
 * Auth       : Authorization: Bearer <token> on every business call.
 * Health     : GET /healthz and GET /readyz — no auth required.
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface PageInfo {
  page: number
  pageSize: number
  total: number
}

export interface ListResponse<T> {
  items: T[]
  page: PageInfo
}

// ---------------------------------------------------------------------------
// Error
// ---------------------------------------------------------------------------

export class ApiError extends Error {
  code: string
  requestId: string
  fields?: Record<string, string>

  constructor(code: string, message: string, requestId: string, fields?: Record<string, string>) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.requestId = requestId
    this.fields = fields
  }
}

// ---------------------------------------------------------------------------
// Envelope shapes (internal)
// ---------------------------------------------------------------------------

interface SuccessEnvelope<T> {
  data: T
  requestId: string
}

interface ListEnvelope<T> {
  data: T[]
  page: PageInfo
  requestId: string
}

interface ErrorEnvelope {
  error: {
    code: string
    message: string
    requestId: string
    fields?: Record<string, string>
  }
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

const AUTH_TOKEN_KEY = 'auth_token'

class ApiClientImpl {
  baseUrl: string
  private token: string | null = null

  constructor() {
    this.baseUrl =
      ((import.meta as Record<string, unknown>).env?.VITE_API_BASE_URL as string | undefined) ??
      '/api/v1'

    // Restore token from localStorage on init
    try {
      const stored = localStorage.getItem(AUTH_TOKEN_KEY)
      if (stored) this.token = stored
    } catch {
      // localStorage may be unavailable (SSR, test env)
    }
  }

  // ── Token management ──

  getToken(): string | null {
    return this.token
  }

  setToken(token: string | null): void {
    this.token = token
    if (token) {
      try {
        localStorage.setItem(AUTH_TOKEN_KEY, token)
      } catch {
        // noop
      }
    } else {
      try {
        localStorage.removeItem(AUTH_TOKEN_KEY)
      } catch {
        // noop
      }
    }
  }

  // ── Request helpers ──

  /**
   * Single-resource request.
   * Parses `{ data, requestId }` on 2xx → returns `data`.
   * Parses `{ error }` on non-2xx → throws `ApiError`.
   * Clears token on 401.
   */
  async doRequest<T>(path: string, options?: RequestInit): Promise<T> {
    const res = await this.fetchWithAuth(path, options)

    if (res.status === 401) {
      this.setToken(null)
    }

    if (!res.ok) {
      throw await this.parseError(res)
    }

    // 204 No Content
    if (res.status === 204) {
      return undefined as T
    }

    const json: SuccessEnvelope<T> = (await res.json()) as SuccessEnvelope<T>
    return json.data
  }

  /**
   * Paginated-list request.
   * Parses `{ data, page, requestId }` on 2xx.
   */
  async listRequest<T>(path: string, options?: RequestInit): Promise<ListResponse<T>> {
    const res = await this.fetchWithAuth(path, options)

    if (res.status === 401) {
      this.setToken(null)
    }

    if (!res.ok) {
      throw await this.parseError(res)
    }

    const json: ListEnvelope<T> = (await res.json()) as ListEnvelope<T>
    return {
      items: json.data,
      page: json.page,
    }
  }

  /**
   * Raw fetch that returns the Response object for binary downloads, etc.
   * Does NOT parse the envelope — caller is responsible for reading body.
   */
  async rawRequest(path: string, options?: RequestInit): Promise<Response> {
    const res = await this.fetchWithAuth(path, options)

    if (res.status === 401) {
      this.setToken(null)
    }

    if (!res.ok) {
      throw await this.parseError(res)
    }

    return res
  }

  // ── Health (no auth) ──

  async healthz(): Promise<{ status: string }> {
    const res = await fetch(`${this.baseUrl}/healthz`)
    if (!res.ok) {
      throw new ApiError('health_check_failed', 'Health check failed', '')
    }
    const json = (await res.json()) as SuccessEnvelope<{ status: string }>
    return json.data
  }

  async readyz(): Promise<{ status: string }> {
    const res = await fetch(`${this.baseUrl}/readyz`)
    if (!res.ok) {
      throw new ApiError('readiness_check_failed', 'Readiness check failed', '')
    }
    const json = (await res.json()) as SuccessEnvelope<{ status: string }>
    return json.data
  }

  // ── Internals ──

  private async fetchWithAuth(path: string, options?: RequestInit): Promise<Response> {
    const headers = new Headers(options?.headers)

    // Ensure Content-Type for requests with a body (unless it's FormData)
    const hasBody = options?.body != null && !(options.body instanceof FormData)
    if (hasBody && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json')
    }

    // Attach auth token for business API calls
    if (this.token) {
      headers.set('Authorization', `Bearer ${this.token}`)
    }

    return fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers,
    })
  }

  private async parseError(res: Response): Promise<ApiError> {
    try {
      const json: ErrorEnvelope = (await res.json()) as ErrorEnvelope
      return new ApiError(
        json.error.code,
        json.error.message,
        json.error.requestId,
        json.error.fields,
      )
    } catch {
      return new ApiError('http_error', `HTTP ${res.status}: ${res.statusText}`, '')
    }
  }
}

/** Singleton API client instance. */
export const apiClient = new ApiClientImpl()

// ---------------------------------------------------------------------------
// Standalone function exports — convenience wrappers for API module imports.
// Usage: import { doRequest, listRequest } from './client'
// ---------------------------------------------------------------------------

/**
 * Single-resource request via the singleton API client.
 * @see ApiClientImpl.doRequest
 */
export function doRequest<T>(path: string, options?: RequestInit): Promise<T> {
  return apiClient.doRequest<T>(path, options)
}

/**
 * Paginated-list request via the singleton API client.
 * @see ApiClientImpl.listRequest
 */
export function listRequest<T>(path: string, options?: RequestInit): Promise<ListResponse<T>> {
  return apiClient.listRequest<T>(path, options)
}
