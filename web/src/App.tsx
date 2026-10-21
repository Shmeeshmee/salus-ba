import { HealthStatus } from './HealthStatus'
import { newApiClient } from './api/client'

const api = newApiClient()

export default function App() {
  return (
    <main>
      <h1>Salus</h1>
      <HealthStatus client={api} />
    </main>
  )
}
