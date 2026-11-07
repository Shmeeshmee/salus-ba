import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.tsx'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { I18nProvider } from './i18n/I18nProvider'
import { startLanguage } from './i18n/saved'
import './tokens.css'
import './app.css'

const browser = navigator.languages.length > 0 ? navigator.languages : [navigator.language]

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <I18nProvider initial={startLanguage(browser)} browser={browser}>
      <App api={newApiClient()} fhir={newFhirClient()} />
    </I18nProvider>
  </StrictMode>,
)
