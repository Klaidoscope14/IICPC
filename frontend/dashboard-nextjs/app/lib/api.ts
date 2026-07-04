// Centralized API client — single source of truth for the backend URL.
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082'

/** Helper to read a cookie value by name (works in browser only). */
export function getCookie(name: string): string | null {
  if (typeof document === 'undefined') return null;
  const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
  return match ? decodeURIComponent(match[1]) : null;
}

/**
 * Wrapper around fetch that prepends the API base URL.
 * Throws on non-ok responses with the parsed error message.
 */
export async function apiClient<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const token = getCookie('token');
  const headers = new Headers(options?.headers || {});
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers,
  })

  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    // Handle gateway-style error objects: {"error": {"code":"...", "message":"..."}}
    const errorField = body.error;
    const message = typeof errorField === 'object' && errorField !== null
      ? errorField.message
      : errorField;
    throw new Error(message || `Request failed with status ${response.status}`)
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
