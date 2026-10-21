import { act } from '@testing-library/react'
import { StrictMode } from 'react'
import { hydrateRoot } from 'react-dom/client'
import { describe, expect, it, vi } from 'vitest'
import template from '../index.html?raw'
import { App } from './App'
import { demo } from './content/demo'
import { product } from './content/features'
import { pricing } from './content/pricing'
import { standards } from './content/standards'
import { howItWorks } from './content/steps'
import { render } from './entry-server'
import { injectApp } from './lib/injectApp'

// The same steps scripts/prerender.ts runs on the built index.html.
const html = injectApp(template, render())
const page = new DOMParser().parseFromString(html, 'text/html')
const text = (selector: string) => [...page.querySelectorAll(selector)].map((el) => el.textContent)

describe('prerendered page', () => {
  it('contains the headline and every section heading, readable without JavaScript', () => {
    expect(text('h1')).toEqual(['Your doctor, one call away.'])
    expect(text('h2')).toEqual([product.heading, howItWorks.heading, standards.heading, pricing.heading, demo.heading])
    expect(page.getElementById('root')?.textContent).toContain(product.features[0].body)
  })

  it('has no undefined anywhere', () => {
    expect(html).not.toContain('undefined')
  })

  it('is hydrated by the client without a mismatch', async () => {
    const container = document.body.appendChild(document.createElement('div'))
    container.innerHTML = render()
    const onRecoverableError = vi.fn()

    await act(async () => {
      hydrateRoot(
        container,
        <StrictMode>
          <App />
        </StrictMode>,
        { onRecoverableError },
      )
    })

    expect(onRecoverableError).not.toHaveBeenCalled()
    container.remove()
  })
})
