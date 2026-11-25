import { useId } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { useLoad } from './load'

type Device = components['schemas']['Device']
type Group = { id: string; name: string; devices: Device[] }

// devicesByLocation reads the clinic's locations in use and the devices of each, in the locations' order.
async function devicesByLocation(api: ApiClient): Promise<Group[]> {
  const sites = await attempt(api.GET('/clinic/locations'))
  if (!sites.ok) throw new Error('locations')
  const active = sites.data.filter((l) => l.active)
  const lists = await Promise.all(active.map((l) => attempt(api.GET('/clinic/locations/{id}/devices', { params: { path: { id: l.id } } }))))
  return active.map((l, i) => {
    const res = lists[i]
    return { id: l.id, name: l.name, devices: res.ok ? res.data : [] }
  })
}

type Props = { api: ApiClient; value?: string; onChange: (deviceId: string | undefined) => void }

// ServiceDevice picks the device a visit for a service needs (U4 of docs/clinic-cms/WAVE6.md): none by default, or a
// device in use of a location in use, grouped by location. The device the service already names stays offered when
// it is out of use, marked so. With no device to offer, or when they cannot be read, nothing is shown and the
// service keeps the device it has.
export function ServiceDevice({ api, value, onChange }: Props) {
  const t = useT()
  const id = useId()
  const { state } = useLoad(() => devicesByLocation(api), [api])
  if (state.name !== 'loaded') return null
  const groups = state.value
    .map((g) => ({ ...g, devices: g.devices.filter((d) => d.active || d.id === value) }))
    .filter((g) => g.devices.length > 0)
  if (groups.length === 0) return null
  return (
    <div className="field">
      <label htmlFor={id}>{t('prices.device')}</label>
      <select id={id} value={value ?? ''} aria-describedby={`${id}-hint`} onChange={(e) => onChange(e.target.value || undefined)}>
        <option value="">{t('prices.deviceNone')}</option>
        {groups.map((g) => (
          <optgroup key={g.id} label={g.name}>
            {g.devices.map((d) => (
              <option key={d.id} value={d.id}>
                {d.active ? d.name : t('prices.deviceInactive', { name: d.name })}
              </option>
            ))}
          </optgroup>
        ))}
      </select>
      <p className="hint" id={`${id}-hint`}>
        {t('prices.deviceHint')}
      </p>
    </div>
  )
}
