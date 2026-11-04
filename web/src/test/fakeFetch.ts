import { expect } from 'vitest'

export type Route = (request: Request) => Response | Promise<Response>

// fakeFetch answers requests from routes keyed by "METHOD /path" and keeps a copy of every request. A request
// with no route fails the test.
export function fakeFetch(routes: Record<string, Route>) {
  const requests: Request[] = []
  const fetch = async (request: Request): Promise<Response> => {
    requests.push(request.clone())
    const key = `${request.method} ${new URL(request.url).pathname}`
    const route = routes[key]
    if (!route) throw new Error(`no fake route for ${key}`)
    return route(request)
  }
  return { fetch, requests }
}

export function apiError(status: number, error: string, headers?: Record<string, string>): Response {
  return Response.json({ error }, { status, headers })
}

export function fhirJSON(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/fhir+json' } })
}

export function outcome(status: number, code: string): Response {
  return fhirJSON({ resourceType: 'OperationOutcome', issue: [{ severity: 'error', code }] }, status)
}

// expectCsrfOnWrites asserts that every request that is not a read carried X-Salus-CSRF: 1.
export function expectCsrfOnWrites(requests: Request[]) {
  const writes = requests.filter((r) => r.method !== 'GET' && r.method !== 'HEAD')
  expect(writes.length).toBeGreaterThan(0)
  for (const r of writes) expect(r.headers.get('X-Salus-CSRF'), `${r.method} ${r.url}`).toBe('1')
}

export function paths(requests: Request[]): string[] {
  return requests.map((r) => `${r.method} ${new URL(r.url).pathname}`)
}
