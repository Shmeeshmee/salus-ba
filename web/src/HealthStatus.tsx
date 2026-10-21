import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'

type Status = 'checking' | 'ok' | 'unavailable'

export function HealthStatus({ client }: { client: ApiClient }) {
  const [status, setStatus] = useState<Status>('checking')

  useEffect(() => {
    let current = true
    client.GET('/healthz').then(
      ({ data }) => current && setStatus(data?.status === 'ok' ? 'ok' : 'unavailable'),
      () => current && setStatus('unavailable'),
    )
    return () => {
      current = false
    }
  }, [client])

  return <p>{`API status: ${status}`}</p>
}
