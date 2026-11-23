import { afterEach, expect, it } from 'vitest'
import appCSS from './app.css?raw'

afterEach(() => {
  document.head.innerHTML = ''
  document.body.innerHTML = ''
})

const look = (el: Element) => {
  const css = getComputedStyle(el)
  return [css.width, css.padding, css.borderTop, css.borderRadius, css.backgroundColor, css.font]
}

it('styles date and tel inputs anywhere like a text input, the desk and the receipts period included', () => {
  const style = document.createElement('style')
  style.textContent = appCSS
  document.head.append(style)
  document.body.innerHTML = '<form><input type="text"><input type="date"><input type="tel"></form>'
  const [text, date, tel] = Array.from(document.querySelectorAll('input'))

  expect(look(text)[1]).toBe('10px 12px')
  expect(look(date)).toEqual(look(text))
  expect(look(tel)).toEqual(look(text))
})

it('keeps no input styling of its own for the patient picker', () => {
  expect(appCSS).not.toMatch(/\.patient-picker input/)
})
