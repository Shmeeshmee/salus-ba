import { StrictMode } from 'react'
import { renderToString } from 'react-dom/server'
import { App } from './App'

// Used only by the prerender (scripts/prerender.ts) and the dev server; never shipped to the browser.
export function render(): string {
  return renderToString(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}
