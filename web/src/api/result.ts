import type { ErrorCode } from './client'

export type Outcome<T> = { ok: true; data: T; response: Response } | { ok: false; code?: ErrorCode; response?: Response }

// attempt turns an openapi-fetch call into one result: the data, or the API's error code (none when the API
// could not be reached or answered something else).
export async function attempt<T>(request: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<Outcome<T>> {
  try {
    const { data, error, response } = await request
    if (response.ok) return { ok: true, data: data as T, response }
    return { ok: false, code: errorCode(error), response }
  } catch {
    return { ok: false }
  }
}

function errorCode(error: unknown): ErrorCode | undefined {
  if (typeof error !== 'object' || error === null || !('error' in error)) return undefined
  return (error as { error: ErrorCode }).error
}
