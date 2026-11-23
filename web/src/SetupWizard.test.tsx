import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { SetupBanner, SetupWizard } from './SetupWizard'
import { apiError, expectCsrfOnWrites, fakeFetch, paths, type Route } from './test/fakeFetch'

afterEach(() => {
  cleanup()
  localStorage.clear()
})

const allOpen = { details: 'open', billing: 'open', hours: 'open', practitioner: 'open', services: 'open' }

function renderWizard(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  renderInLanguage(<SetupWizard api={newApiClient(fake.fetch)} />, 'en')
  return fake
}

describe('SetupWizard', () => {
  it('lists the five steps in order, each linking to its own screen', async () => {
    renderWizard({ 'GET /clinic/setup': () => Response.json({ ...allOpen, details: 'done' }) })

    const items = await screen.findAllByRole('listitem')
    expect(items.map((li) => within(li).getByRole('heading').textContent)).toEqual([
      'Clinic details and address',
      'Billing details',
      'Working hours',
      'Invite the first practitioner',
      'Price list',
    ])
    expect(within(items[0]).getByText('Done')).toBeTruthy()
    expect(within(items[1]).getByText('To do')).toBeTruthy()
    expect(items.map((li) => within(li).getByRole('link', { name: 'Open' }).getAttribute('href'))).toEqual([
      '/clinic',
      '/membership',
      '/availability',
      '/practitioners',
      '/price-list',
    ])
  })

  it('saves a step skipped, keeping the others as they are', async () => {
    const { requests } = renderWizard({
      'GET /clinic/setup': () => Response.json(allOpen),
      'PUT /clinic/setup': async (r) => Response.json(await r.json()),
    })

    const hours = (await screen.findAllByRole('listitem'))[2]
    fireEvent.click(within(hours).getByRole('button', { name: 'Skip' }))

    expect(await within(hours).findByText('Skipped')).toBeTruthy()
    expect(paths(requests)).toEqual(['GET /clinic/setup', 'PUT /clinic/setup'])
    expect(await requests[1].json()).toEqual({ ...allOpen, hours: 'skipped' })
    expectCsrfOnWrites(requests)
    fireEvent.click(within(hours).getByRole('button', { name: 'Reopen' }))
    expect(await within(hours).findByText('To do')).toBeTruthy()
  })

  it('says when the progress cannot be read or saved', async () => {
    renderWizard({ 'GET /clinic/setup': () => Response.json(allOpen), 'PUT /clinic/setup': () => apiError(500, 'internal error') })

    fireEvent.click(within((await screen.findAllByRole('listitem'))[0]).getByRole('button', { name: 'Mark as done' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Something went wrong. Try again.')
  })
})

describe('SetupBanner', () => {
  it('shows while a step is open and links to the setup', async () => {
    const fake = fakeFetch({ 'GET /clinic/setup': () => Response.json({ ...allOpen, details: 'done' }) })
    renderInLanguage(<SetupBanner api={newApiClient(fake.fetch)} />, 'en')

    expect(await screen.findByText("Your clinic's setup is not finished.")).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Continue setup' }).getAttribute('href')).toBe('/setup')
  })

  it('is hidden when every step is done or skipped', async () => {
    const fake = fakeFetch({
      'GET /clinic/setup': () => Response.json({ details: 'done', billing: 'skipped', hours: 'done', practitioner: 'done', services: 'skipped' }),
    })
    const { container } = renderInLanguage(<SetupBanner api={newApiClient(fake.fetch)} />, 'en')

    await new Promise((r) => setTimeout(r, 0))
    expect(container.textContent).toBe('')
    expect(fake.requests).toHaveLength(1)
  })
})
