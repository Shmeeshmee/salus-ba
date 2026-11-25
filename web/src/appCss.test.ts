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

// The rules computed in jsdom: the walk's layout findings (docs/clinic-cms/walk/wave3, findings 4 to 7).
function styled(html: string) {
  const style = document.createElement('style')
  style.textContent = appCSS
  document.head.append(style)
  document.body.innerHTML = html
}

it("lays the walk-in's search out as the form below it: one column, the dialog's full width", () => {
  styled('<div class="receipt-dialog desk-dialog"><div class="patient-picker"><form class="toolbar" role="search"><div class="field"></div><button></button></form></div></div>')
  const form = getComputedStyle(document.querySelector('form')!)
  expect(form.display).toBe('grid')
  expect(form.gridTemplateColumns).toBe('minmax(0px, 1fr)')
  expect(getComputedStyle(document.querySelector('form button')!).width).toBe('100%')
})

it("puts the reversal's label above its select", () => {
  styled('<div class="confirm reverse"><label><span></span><select></select></label></div>')
  expect(getComputedStyle(document.querySelector('label')!).display).toBe('grid')
})

it("lays the visit page's actions out in one wrapping row, two even columns on a phone", () => {
  expect(appCSS).toMatch(/\.visit-actions \.actions \{[^}]*flex-wrap: wrap/)
  expect(appCSS).toMatch(/@media \(max-width: 520px\) \{[^@]*\.visit-actions \.actions \{ display: grid; grid-template-columns: 1fr 1fr;/)
})

// The wave 6 polish walk (docs/clinic-cms/walk/wave6-polish).
it("lays the desk's patient search out as the walk-in's: one column, the dialog's full width", () => {
  styled('<div class="receipt-dialog desk-dialog"><form class="toolbar" role="search"><div class="field"></div><button></button></form></div>')
  const form = getComputedStyle(document.querySelector('form')!)
  expect(form.display).toBe('grid')
  expect(form.gridTemplateColumns).toBe('minmax(0px, 1fr)')
  expect(getComputedStyle(document.querySelector('form button')!).width).toBe('100%')
})

it("puts a patient's name on the left in the patient search's results, as on the desk's rows", () => {
  styled('<ul class="rows"><li><span class="stack"><button class="desk-name"></button><span></span></span></li></ul>')
  const name = getComputedStyle(document.querySelector('.desk-name')!)
  expect(name.justifySelf).toBe('start')
  expect(name.textAlign).toBe('left')
})
