import { describe, expect, it } from 'vitest'
import { appHref } from './appHref'

describe('appHref', () => {
  it('links to the path on the web app set at build time', () => {
    expect(appHref('/signup', 'https://app.example.test')).toBe('https://app.example.test/signup')
    expect(appHref('/signup', ' https://app.example.test/ ')).toBe('https://app.example.test/signup')
  })

  // Unset, the link shows the placeholder like [PRICE], never a relative path that breaks on another host.
  it('shows the [APP URL] placeholder while the address is not set', () => {
    expect(appHref('/signup', undefined)).toBe('[APP URL]/signup')
    expect(appHref('/signup', '  ')).toBe('[APP URL]/signup')
  })
})
