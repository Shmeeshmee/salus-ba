import { afterEach, expect, it } from 'vitest'
import analyticsCSS from './analytics/analytics.css?raw'
import appCSS from './app.css?raw'
import chatCSS from './chat/chat.css?raw'
import cheatsheetCSS from './terminology/cheatsheet.css?raw'
import explainCSS from './terminology/explain.css?raw'
import reviewCSS from './terminology/review.css?raw'

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

// RD6: a disabled button looks disabled, whatever its kind: one flat grey look, not a faded primary. jsdom drops a
// declaration that uses var(), so the colours are read from the rule and the rest computed.
it('gives every disabled button one grey look, and an enabled primary its own', () => {
  styled(
    '<button class="primary" disabled>a</button><div class="actions"><button disabled>b</button><button class="danger" disabled>c</button></div>' +
      '<button class="primary">e</button>',
  )
  const [primary, plain, danger, enabled] = Array.from(document.querySelectorAll('button')).map((b) => getComputedStyle(b))
  const look = (c: CSSStyleDeclaration) => [c.opacity, c.cursor]
  expect(look(primary)).toEqual(['1', 'not-allowed'])
  expect(look(plain)).toEqual(look(primary))
  expect(look(danger)).toEqual(look(primary))
  expect(look(enabled)).toEqual(['1', 'pointer'])
  const rule = appCSS.match(/button\.primary:disabled, [^{]*\{([^}]*)\}/)?.[1] ?? ''
  expect(rule).toContain('background: var(--color-disabled);')
  expect(rule).toContain('color: var(--color-disabled-text);')
  expect(rule).toContain('box-shadow: inset 0 0 0 1px var(--color-disabled-border);')
  expect(appCSS).not.toMatch(/button\.primary:disabled \{ opacity/)
})

// WAVE7 RD3: the keyboard's focus is seen at 3:1 or more, and a finger's targets are 44 px (test/e2e/web/a11y.spec.ts).
const focusCSS = [analyticsCSS, appCSS, chatCSS, cheatsheetCSS, explainCSS, reviewCSS]

it('draws every focus ring in the brand green, not the mint, which is 1.9:1 on white', () => {
  const rings = focusCSS.flatMap((css) => css.match(/[^{}]*:focus[^{]*\{[^}]*outline:[^}]*\}/g) ?? [])
  expect(rings.length).toBeGreaterThan(0)
  for (const r of rings) expect(r).not.toMatch(/outline: 2px solid var\(--color-mint\)/)
  expect(appCSS).toMatch(/input:focus-visible, button:focus-visible[^{]*\{ outline: 2px solid var\(--color-brand\)/)
})

it('keeps a focus ring on the medical terms search', () => {
  expect(cheatsheetCSS).toMatch(/\.cheatsheet-search input/)
  expect(cheatsheetCSS).not.toMatch(/\.cheatsheet-search input:focus-visible \{ outline: none/)
})

it('gives every control at least 44 px on a touch screen', () => {
  const touch = appCSS.match(/@media \(pointer: coarse\) \{([\s\S]*?)\n\}/)?.[1] ?? ''
  expect(touch).toMatch(/button, select,[^{]*\{[^}]*min-height: 44px/)
  expect(touch).toMatch(/button\.icon[^{]*\{[^}]*width: 44px; height: 44px/)
  expect(touch).toMatch(/\.check label, \.choice label[^{]*\{[^}]*min-height: 44px/)
})

it("keeps the focused heatmap cell's 2 px ring when it is also the cell shown", () => {
  expect(analyticsCSS).toMatch(/\.chart-svg \.cell\.active \{[^}]*\}[\s\S]*\.chart-svg \.cell\.active:focus-visible \{ stroke-width: 2; \}/)
})
