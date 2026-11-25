import { useState } from 'react'
import type { ApiClient, ErrorCode } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useLanguage, useT } from './i18n/useT'
import { useLoad } from './load'
import { LocationResources } from './LocationResources'
import { groupPhone } from './phone'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Location = components['schemas']['Location']
type SaveRequest = components['schemas']['SaveLocationRequest']
type Place = components['schemas']['Place']
type T = ReturnType<typeof useT>

// Locations is the admin's screen of the clinic's sites (U1 of docs/clinic-cms/WAVE6.md, package LC2): each with its
// name, address, city and phone, and whether it is in use; under each, its rooms and devices (LocationResources). A
// location is never deleted: taken out of use, it is off the booking and past visits keep its address. One location is
// edited at a time, saved at the version it was read at (If-Match). The time zone is the clinic's: the form keeps the
// one a location has and sends none for a new one.
export function Locations({ api }: { api: ApiClient }) {
  const t = useT()
  const { language } = useLanguage()
  const [editing, setEditing] = useState<string>() // a location's id, or 'new'
  const [opened, setOpened] = useState<string>() // the location whose rooms and devices are shown
  const [saved, setSaved] = useState(false)
  const { state, set } = useLoad(async () => {
    const [res, placesRes] = await Promise.all([attempt(api.GET('/clinic/locations')), attempt(api.GET('/places'))])
    if (!res.ok) throw new Error('locations')
    return { locations: res.data, places: placesRes.ok ? placesRes.data.places : [] }
  }, [api])
  const loaded = state.name === 'loaded' ? state.value : undefined
  const locations = loaded?.locations ?? []
  const places = loaded?.places ?? []
  const name = language === 'sr-Cyrl' ? (p: Place) => p.nameCyrl : (p: Place) => p.nameLatn
  const open = (id: string | undefined) => {
    setEditing(id)
    setSaved(false)
  }
  const stored = (l: Location) => {
    if (!loaded) return
    set({ ...loaded, locations: [...locations.filter((x) => x.id !== l.id), l] })
    setEditing(undefined)
    setSaved(true)
  }
  const city = (id?: string) => places.find((p) => p.id === id)
  return (
    <section className="panel locations">
      <h1>{t('locations.title')}</h1>
      <p className="hint">{t('locations.intro')}</p>
      {state.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
      {state.name === 'failed' && <Alert message={t('locations.loadFailed')} />}
      {saved && <Notice message={t('locations.saved')} />}
      <ul className="rows">
        {locations.map((l) => (
          <li key={l.id} className="location">
            {editing === l.id ? (
              <LocationForm api={api} location={l} places={places} name={name} onStored={stored} onCancel={() => open(undefined)} />
            ) : (
              <>
                <div className="stack">
                  <span className="strong">{l.name}</span>
                  <span className="muted small">{address(l, city(l.cityId), name)}</span>
                </div>
                <div className="actions">
                  {!l.active && <span className="tag">{t('locations.inactive')}</span>}
                  <button type="button" aria-expanded={opened === l.id} aria-label={t('locations.resourcesOf', { name: l.name })} onClick={() => setOpened(opened === l.id ? undefined : l.id)}>
                    {t('locations.resources')}
                  </button>
                  <button type="button" aria-label={t('locations.editNamed', { name: l.name })} onClick={() => open(l.id)}>
                    {t('locations.edit')}
                  </button>
                </div>
                {opened === l.id && <LocationResources api={api} locationId={l.id} />}
              </>
            )}
          </li>
        ))}
      </ul>
      {loaded && editing === 'new' && <LocationForm api={api} places={places} name={name} onStored={stored} onCancel={() => open(undefined)} />}
      {loaded && editing === undefined && (
        <div className="actions">
          <button type="button" onClick={() => open('new')}>
            {t('locations.add')}
          </button>
        </div>
      )}
    </section>
  )
}

// address is a location's street, postal code and city, then its phone in groups; what it has of them.
function address(l: Location, city: Place | undefined, name: (p: Place) => string): string {
  const town = [l.postalCode, city && name(city)].filter(Boolean).join(' ')
  const street = [l.addressLine, town].filter(Boolean).join(', ')
  return [street, l.phone && groupPhone(l.phone)].filter(Boolean).join(' · ')
}

type FormProps = {
  api: ApiClient
  location?: Location
  places: Place[]
  name: (p: Place) => string
  onStored: (l: Location) => void
  onCancel: () => void
}

function LocationForm({ api, location, places, name, onStored, onCancel }: FormProps) {
  const t = useT()
  const [title, setTitle] = useState(location?.name ?? '')
  const [street, setStreet] = useState(location?.addressLine ?? '')
  const [postal, setPostal] = useState(location?.postalCode ?? '')
  const [cityId, setCityId] = useState(location?.cityId ?? '')
  const [phone, setPhone] = useState(groupPhone(location?.phone ?? ''))
  const [active, setActive] = useState(location?.active ?? true)
  const { busy, error, onSubmit } = useSubmit(async () => {
    if (title.trim() === '') return t('locations.invalidName')
    const optional = (key: keyof SaveRequest, value: string) => (value.trim() === '' ? {} : { [key]: value.trim() })
    const body: SaveRequest = {
      name: title.trim(),
      ...optional('addressLine', street),
      ...optional('postalCode', postal),
      ...optional('cityId', cityId),
      ...optional('phone', phone.replace(/\s/g, '')),
      ...(location ? { timeZone: location.timeZone } : {}),
      active,
    }
    const res = location
      ? await attempt(api.PUT('/clinic/locations/{id}', { params: { path: { id: location.id }, header: { 'If-Match': `W/"${location.versionId}"` } }, body }))
      : await attempt(api.POST('/clinic/locations', { body }))
    if (!res.ok) return saveMessage(t, res.response?.status, res.code)
    onStored(res.data)
    return undefined
  })
  const byName = (a: Place, b: Place) => name(a).localeCompare(name(b))
  const regions = places.filter((p) => p.kind === 'region').sort(byName)
  return (
    <form onSubmit={onSubmit} noValidate aria-label={location ? t('locations.editNamed', { name: location.name }) : t('locations.add')}>
      <Field label={t('locations.name')} maxLength={200} value={title} onChange={(e) => setTitle(e.target.value)} />
      <Field label={t('locations.street')} maxLength={200} value={street} onChange={(e) => setStreet(e.target.value)} />
      <div className="pair">
        <Field label={t('locations.postalCode')} maxLength={20} value={postal} onChange={(e) => setPostal(e.target.value)} />
        <div className="field">
          <label htmlFor="location-city">{t('locations.city')}</label>
          <select id="location-city" value={cityId} onChange={(e) => setCityId(e.target.value)}>
            <option value="">{t('locations.noCity')}</option>
            {regions.map((r) => (
              <optgroup key={r.id} label={name(r)}>
                {places
                  .filter((p) => p.kind === 'city' && p.parent === r.id)
                  .sort(byName)
                  .map((p) => (
                    <option key={p.id} value={p.id}>
                      {name(p)}
                    </option>
                  ))}
              </optgroup>
            ))}
          </select>
        </div>
      </div>
      <Field label={t('locations.phone')} type="tel" autoComplete="off" value={phone} onChange={(e) => setPhone(e.target.value)} />
      <label className="check">
        <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} />
        <span>{t('locations.active')}</span>
      </label>
      <p className="hint">{t('locations.activeHint')}</p>
      <Alert message={error} />
      <div className="actions">
        <Submit busy={busy} label={t('locations.save')} busyLabel={t('common.saving')} />
        <button type="button" onClick={onCancel} disabled={busy}>
          {t('locations.cancel')}
        </button>
      </div>
    </form>
  )
}

function saveMessage(t: T, status: number | undefined, code: ErrorCode | undefined): string {
  if (status === 412) return t('locations.conflict')
  if (code === 'forbidden') return t('locations.forbidden')
  if (code === 'invalid_location' || code === 'invalid_request') return t('locations.invalidLocation')
  return t('locations.saveFailed')
}
