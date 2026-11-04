import type { Fetch } from './client'

// Hand-written types for the FHIR R4 elements the web app uses; everything else is absent.
export interface Meta {
  versionId: string
  lastUpdated: string
}

export interface Reference {
  reference: string
}

export interface Organization {
  resourceType: 'Organization'
  id: string
  meta: Meta
  name: string
  active: boolean
}

export interface HumanName {
  family?: string
  given?: string[]
}

export interface Practitioner {
  resourceType: 'Practitioner'
  id: string
  meta: Meta
  name?: HumanName[]
  active: boolean
}

export interface PractitionerRole {
  resourceType: 'PractitionerRole'
  id: string
  meta: Meta
  practitioner: Reference
  organization: Reference
  active: boolean
}

export interface Bundle {
  resourceType: 'Bundle'
  type: 'searchset'
  total?: number
  entry?: { fullUrl: string; resource: PractitionerRole | Practitioner; search?: { mode: 'match' | 'include' } }[]
}

// FhirError is a FHIR call that did not succeed: the HTTP status and the first OperationOutcome issue code.
export class FhirError extends Error {
  readonly status: number
  readonly code?: string

  constructor(status: number, code?: string) {
    super(`FHIR request failed with ${status}`)
    this.status = status
    this.code = code
  }
}

export type FhirClient = ReturnType<typeof newFhirClient>

const fhirJSON = 'application/fhir+json'

// newFhirClient calls /fhir/R4 on the page's own origin with the session cookie. Writes carry the CSRF header
// and If-Match with the version that was read.
export function newFhirClient(fetchImpl: Fetch = (r) => fetch(r)) {
  async function send<T>(method: string, path: string, body?: unknown, version?: string): Promise<T> {
    const headers = new Headers({ Accept: fhirJSON })
    if (body !== undefined) {
      headers.set('Content-Type', fhirJSON)
      headers.set('X-Salus-CSRF', '1')
    }
    if (version !== undefined) headers.set('If-Match', `W/"${version}"`)
    const url = new URL(`/fhir/R4/${path}`, globalThis.location.origin)
    const res = await fetchImpl(
      new Request(url, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) }),
    )
    if (!res.ok) throw new FhirError(res.status, await issueCode(res))
    return (await res.json()) as T
  }

  return {
    get: <T>(path: string) => send<T>('GET', path),
    put: <T>(path: string, resource: unknown, version: string) => send<T>('PUT', path, resource, version),
  }
}

async function issueCode(res: Response): Promise<string | undefined> {
  try {
    const body = (await res.json()) as { resourceType?: string; issue?: { code?: string }[] }
    return body.resourceType === 'OperationOutcome' ? body.issue?.[0]?.code : undefined
  } catch {
    return undefined
  }
}
