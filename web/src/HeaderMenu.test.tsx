import { act, cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { HeaderMenu } from './HeaderMenu'
import { renderInLanguage } from './i18n/renderInLanguage'
import { navigate } from './route'
import { Link } from './ui'

afterEach(cleanup)

// The header's menu on a phone (readiness): a disclosure button and the panel it shows. On a wider screen the CSS
// shows the panel's contents in the header and hides the button; jsdom has no layout, so these tests state the
// behaviour of the disclosure itself.
function renderMenu(path = '/') {
  history.replaceState(null, '', path)
  renderInLanguage(
    <header>
      <HeaderMenu>
        <nav aria-label="Main">
          <ul>
            <li>
              <Link to="/">Calendar</Link>
            </li>
            <li>
              <Link to="/waitlist">Waitlist</Link>
            </li>
          </ul>
        </nav>
        <button type="button">Sign out</button>
      </HeaderMenu>
    </header>,
    'en',
  )
  const button = screen.getByRole('button', { name: 'Menu' })
  const panel = document.getElementById(button.getAttribute('aria-controls') ?? '')
  if (!panel) throw new Error('the menu button controls no element')
  return { button, panel }
}

const expanded = (b: HTMLElement) => b.getAttribute('aria-expanded')
const isOpen = (panel: HTMLElement) => panel.classList.contains('open')

describe('HeaderMenu', () => {
  it('is a button that shows and hides the panel it controls, saying whether it is open', () => {
    const { button, panel } = renderMenu()
    expect(button.tagName).toBe('BUTTON')
    expect(button.getAttribute('type')).toBe('button')
    expect(expanded(button)).toBe('false')
    expect(isOpen(panel)).toBe(false)
    expect(within(panel).getAllByRole('link').map((l) => l.textContent)).toEqual(['Calendar', 'Waitlist'])

    fireEvent.click(button)
    expect(expanded(button)).toBe('true')
    expect(isOpen(panel)).toBe(true)

    fireEvent.click(button)
    expect(expanded(button)).toBe('false')
    expect(isOpen(panel)).toBe(false)
  })

  it('comes before the panel, so Tab goes from the button to the panel', () => {
    const { button, panel } = renderMenu()
    expect(button.compareDocumentPosition(panel) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(panel.getAttribute('role')).toBeNull() // a disclosure, not a modal: no dialog, no focus trap
    expect(panel.getAttribute('aria-modal')).toBeNull()
  })

  it('marks the current page in the panel as the header does', () => {
    const { panel } = renderMenu('/waitlist')
    expect(within(panel).getByRole('link', { name: 'Waitlist' }).getAttribute('aria-current')).toBe('page')
    expect(within(panel).getByRole('link', { name: 'Calendar' }).getAttribute('aria-current')).toBeNull()
  })

  it('closes on Escape from inside the panel and gives the focus back to the button', () => {
    const { button, panel } = renderMenu()
    fireEvent.click(button)
    const link = within(panel).getByRole('link', { name: 'Waitlist' })
    link.focus()
    fireEvent.keyDown(link, { key: 'Escape' })
    expect(expanded(button)).toBe('false')
    expect(document.activeElement).toBe(button)
  })

  it('closes on Escape on the button itself', () => {
    const { button } = renderMenu()
    fireEvent.click(button)
    button.focus()
    fireEvent.keyDown(button, { key: 'Escape' })
    expect(expanded(button)).toBe('false')
    expect(document.activeElement).toBe(button)
  })

  it('leaves an Escape that a dialog opened from the panel has handled to that dialog', () => {
    const { button, panel } = renderMenu()
    fireEvent.click(button)
    const signOut = within(panel).getByRole('button', { name: 'Sign out' })
    signOut.addEventListener('keydown', (e) => e.preventDefault())
    fireEvent.keyDown(signOut, { key: 'Escape' })
    expect(expanded(button)).toBe('true')
    cleanup()

    // As the medical terms' palette does it: its own React handler prevents the default.
    renderInLanguage(
      <HeaderMenu>
        <div role="dialog" aria-label="Palette" onKeyDown={(e) => e.key === 'Escape' && e.preventDefault()}>
          <input aria-label="Search" />
        </div>
      </HeaderMenu>,
      'en',
    )
    const menu = screen.getByRole('button', { name: 'Menu' })
    fireEvent.click(menu)
    fireEvent.keyDown(screen.getByRole('textbox', { name: 'Search' }), { key: 'Escape' })
    expect(expanded(menu)).toBe('true')
  })

  it('closes when a link is chosen, the current page included', () => {
    const { button, panel } = renderMenu('/')
    fireEvent.click(button)
    fireEvent.click(within(panel).getByRole('link', { name: 'Waitlist' }))
    expect(location.pathname).toBe('/waitlist')
    expect(expanded(button)).toBe('false')

    fireEvent.click(button)
    fireEvent.click(within(panel).getByRole('link', { name: 'Waitlist' }))
    expect(location.pathname).toBe('/waitlist')
    expect(expanded(button)).toBe('false')
  })

  it('closes when the page changes some other way', () => {
    const { button } = renderMenu('/')
    fireEvent.click(button)
    act(() => navigate('/waitlist'))
    expect(expanded(button)).toBe('false')
  })

  it('stays open while a control that is not a link is used', () => {
    const { button, panel } = renderMenu()
    fireEvent.click(button)
    fireEvent.click(within(panel).getByRole('button', { name: 'Sign out' }))
    expect(expanded(button)).toBe('true')
  })

  it('is named in each staff language', () => {
    const names = { bs: 'Meni', hr: 'Izbornik', 'sr-Latn': 'Meni', 'sr-Cyrl': 'Мени', en: 'Menu' } as const
    for (const [language, name] of Object.entries(names)) {
      renderInLanguage(
        <HeaderMenu>
          <span />
        </HeaderMenu>,
        language as keyof typeof names,
      )
      expect(screen.getByRole('button', { name })).toBeTruthy()
      cleanup()
    }
  })
})
