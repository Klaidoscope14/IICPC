// Centralized API client — single source of truth for the backend URL.
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

/**
 * Wrapper around fetch that prepends the API base URL.
 * Throws on non-ok responses with the parsed error message.
 */
export async function apiClient<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, options)

  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error || `Request failed with status ${response.status}`)
  }

  return response.json()
}

/**
 * POST JSON to the API.
 */
export async function apiPost<T>(path: string, data: unknown): Promise<T> {
  return apiClient<T>(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}
