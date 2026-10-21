import { describe, expect, it } from 'vitest'
import { injectApp } from './injectApp'

describe('injectApp', () => {
  it('puts the rendered app in place of the marker', () => {
    expect(injectApp('<div id="root"><!--app--></div>', '<p>Hi</p>')).toBe('<div id="root"><p>Hi</p></div>')
  })

  it('keeps dollar signs in the app as they are', () => {
    expect(injectApp('<!--app-->', '<p>$& $1</p>')).toBe('<p>$& $1</p>')
  })

  it('fails when the template has no marker', () => {
    expect(() => injectApp('<div id="root"></div>', '<p>Hi</p>')).toThrow('<!--app-->')
  })
})
