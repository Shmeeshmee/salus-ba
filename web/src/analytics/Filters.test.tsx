import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../i18n/I18nProvider'
import type { Language } from '../i18n/languages'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { drillOf } from './drill'
import { DiagnosisFilter, MonthRange, PlaceFilter } from './Filters'
import { fixtureReference } from './fixtures'

afterEach(cleanup)

const { groups, places } = fixtureReference()

// inLanguage wraps ui in the provider, for a render and its rerenders alike.
const inLanguage = (language: Language) => (ui: ReactNode) => (
  <I18nProvider initial={language} browser={[language]}>
    {ui}
  </I18nProvider>
)

const options = (select: HTMLElement) => within(select).getAllByRole('option').map((o) => o.textContent)

describe('MonthRange', () => {
  it('offers the months in the UI language and keeps from before to', () => {
    const onChange = vi.fn()
    renderInLanguage(<MonthRange months={['2026-08', '2026-09', '2026-10']} from="2026-09" to="2026-10" onChange={onChange} />, 'hr')
    expect(options(screen.getByLabelText('Od'))).toEqual(['kolovoz 2026.', 'rujan 2026.', 'listopad 2026.'])
    fireEvent.change(screen.getByLabelText('Do'), { target: { value: '2026-08' } })
    expect(onChange).toHaveBeenCalledWith({ from: '2026-08', to: '2026-08' })
  })
})

describe('DiagnosisFilter and drillOf', () => {
  it('offers a subgroup only for a group that has subgroups', async () => {
    const all = await groups('en')
    const onChange = vi.fn()
    const en = inLanguage('en')
    const { rerender } = render(en(<DiagnosisFilter groups={all} value={{}} onChange={onChange} />))
    expect(screen.queryByLabelText('Subgroup')).toBeNull()
    fireEvent.change(screen.getByLabelText('Diagnosis group'), { target: { value: 'respiratory-infections' } })
    expect(onChange).toHaveBeenCalledWith({ group: 'respiratory-infections' })
    rerender(en(<DiagnosisFilter groups={all} value={{ group: 'respiratory-infections' }} onChange={onChange} />))
    expect(options(screen.getByLabelText('Subgroup'))).toEqual([
      'All of the group',
      'Group common-cold',
      'Group upper-airway-infections',
      'Group flu',
      'Group pneumonia',
      'Group chest-infections',
    ])
    rerender(en(<DiagnosisFilter groups={all} value={{ group: 'diabetes' }} onChange={onChange} />))
    expect(screen.queryByLabelText('Subgroup')).toBeNull()
  })

  it('breaks down by group, then subgroup where there are any, then code', async () => {
    const all = await groups('en')
    expect(drillOf({}, all)).toBe('group')
    expect(drillOf({ group: 'respiratory-infections' }, all)).toBe('subgroup')
    expect(drillOf({ group: 'diabetes' }, all)).toBe('code')
    expect(drillOf({ group: 'respiratory-infections', subgroup: 'flu' }, all)).toBe('code')
  })
})

describe('PlaceFilter', () => {
  it('offers the entity and the regions, then the cities of the region chosen', async () => {
    const all = await places()
    const onChange = vi.fn()
    const cyrl = inLanguage('sr-Cyrl')
    const { rerender } = render(cyrl(<PlaceFilter places={all} value={{}} onChange={onChange} />))
    const region = screen.getByLabelText('Подручје')
    expect(options(region).slice(0, 3)).toEqual(['Цијела БиХ', 'Федерација Босне и Херцеговине', 'Унско-сански кантон'])
    expect(screen.queryByLabelText('Град')).toBeNull()
    fireEvent.change(region, { target: { value: 'BA-09' } })
    expect(onChange).toHaveBeenCalledWith({ place: 'BA-09' })
    rerender(cyrl(<PlaceFilter places={all} value={{ place: 'BA-09' }} onChange={onChange} />))
    expect(options(screen.getByLabelText('Град'))).toEqual(['Сви градови', 'Центар', 'Нови Град', 'Трново'])
    fireEvent.change(screen.getByLabelText('Град'), { target: { value: 'Q-fixture-2' } })
    expect(onChange).toHaveBeenLastCalledWith({ place: 'BA-09', city: 'Q-fixture-2' })
  })

  it('offers no city when cities are not a level of the view', async () => {
    renderInLanguage(<PlaceFilter places={await places()} value={{ place: 'BA-09' }} onChange={() => {}} cities={false} />, 'en')
    expect(screen.queryByLabelText('City')).toBeNull()
  })
})
