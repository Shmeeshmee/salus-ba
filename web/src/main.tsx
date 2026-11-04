import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.tsx'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import './tokens.css'
import './app.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App api={newApiClient()} fhir={newFhirClient()} />
  </StrictMode>,
)
