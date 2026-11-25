import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt, type Outcome } from './api/result'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { useLoad } from './load'
import { Alert, Field, Submit } from './ui'
import { useSubmit } from './useSubmit'

// A room (U3) and a device (U4) have the same shape: a name of one location, in use or not, never deleted or moved.
type Resource = components['schemas']['Room'] | components['schemas']['Device']
type SaveRequest = components['schemas']['SaveRoomRequest']
type Kind = 'room' | 'device'

// The calls of each kind, against the routes of package RM1.
function calls(api: ApiClient, kind: Kind) {
  const header = (r: Resource) => ({ path: { id: r.id }, header: { 'If-Match': `W/"${r.versionId}"` } })
  return kind === 'room'
    ? {
        list: (locationId: string) => attempt(api.GET('/clinic/locations/{id}/rooms', { params: { path: { id: locationId } } })),
        add: (locationId: string, body: SaveRequest) => attempt(api.POST('/clinic/locations/{id}/rooms', { params: { path: { id: locationId } }, body })),
        change: (r: Resource, body: SaveRequest) => attempt(api.PUT('/clinic/rooms/{id}', { params: header(r), body })),
      }
    : {
        list: (locationId: string) => attempt(api.GET('/clinic/locations/{id}/devices', { params: { path: { id: locationId } } })),
        add: (locationId: string, body: SaveRequest) => attempt(api.POST('/clinic/locations/{id}/devices', { params: { path: { id: locationId } }, body })),
        change: (r: Resource, body: SaveRequest) => attempt(api.PUT('/clinic/devices/{id}', { params: header(r), body })),
      }
}

// LocationResources lists a location's rooms, which an in-person rule of working hours may name, and its devices,
// which a service may require; the admin adds them and changes their name or whether they are in use.
export function LocationResources({ api, locationId }: { api: ApiClient; locationId: string }) {
  const t = useT()
  const { state } = useLoad(async () => {
    const [rooms, devices] = await Promise.all([calls(api, 'room').list(locationId), calls(api, 'device').list(locationId)])
    if (!rooms.ok || !devices.ok) throw new Error('resources')
    return { rooms: rooms.data, devices: devices.data }
  }, [api, locationId])
  if (state.name === 'loading') return <p className="muted">{t('common.loading')}</p>
  if (state.name === 'failed') return <Alert message={t('locations.resourcesFailed')} />
  return (
    <div className="location-resources">
      <ResourceList api={api} kind="room" locationId={locationId} initial={state.value.rooms} />
      <ResourceList api={api} kind="device" locationId={locationId} initial={state.value.devices} />
    </div>
  )
}

const labels = {
  room: { title: 'locations.rooms', hint: 'locations.roomsHint', empty: 'locations.noRooms', add: 'locations.addRoom', name: 'locations.roomName', save: 'locations.saveRoom' },
  device: { title: 'locations.devices', hint: 'locations.devicesHint', empty: 'locations.noDevices', add: 'locations.addDevice', name: 'locations.deviceName', save: 'locations.saveDevice' },
} as const

type ListProps = { api: ApiClient; kind: Kind; locationId: string; initial: Resource[] }

function ResourceList({ api, kind, locationId, initial }: ListProps) {
  const t = useT()
  const l = labels[kind]
  const [items, setItems] = useState(initial)
  const [editing, setEditing] = useState<string>() // an item's id, or 'new'
  const stored = (r: Resource) => {
    setItems([...items.filter((x) => x.id !== r.id), r].sort((a, b) => a.name.localeCompare(b.name)))
    setEditing(undefined)
  }
  const form = (item?: Resource) => (
    <ResourceForm
      kind={kind}
      item={item}
      save={(body) => (item ? calls(api, kind).change(item, body) : calls(api, kind).add(locationId, body))}
      onStored={stored}
      onCancel={() => setEditing(undefined)}
    />
  )
  return (
    <div className="location-resource">
      <h3 id={`${kind}s-${locationId}`}>{t(l.title)}</h3>
      <p className="hint">{t(l.hint)}</p>
      {items.length === 0 && <p className="muted small">{t(l.empty)}</p>}
      <ul className="rows" aria-labelledby={`${kind}s-${locationId}`}>
        {items.map((r) => (
          <li key={r.id}>
            {editing === r.id ? (
              form(r)
            ) : (
              <>
                <span>{r.name}</span>
                <div className="actions">
                  {!r.active && <span className="tag">{t('locations.inactive')}</span>}
                  <button type="button" aria-label={t('locations.editNamed', { name: r.name })} onClick={() => setEditing(r.id)}>
                    {t('locations.edit')}
                  </button>
                </div>
              </>
            )}
          </li>
        ))}
      </ul>
      {editing === 'new' && form()}
      {editing === undefined && (
        <div className="actions">
          <button type="button" onClick={() => setEditing('new')}>
            {t(l.add)}
          </button>
        </div>
      )}
    </div>
  )
}

type FormProps = {
  kind: Kind
  item?: Resource
  save: (body: SaveRequest) => Promise<Outcome<Resource>>
  onStored: (r: Resource) => void
  onCancel: () => void
}

function ResourceForm({ kind, item, save, onStored, onCancel }: FormProps) {
  const t = useT()
  const l = labels[kind]
  const [name, setName] = useState(item?.name ?? '')
  const [active, setActive] = useState(item?.active ?? true)
  const { busy, error, onSubmit } = useSubmit(async () => {
    if (name.trim() === '') return t('locations.invalidName')
    const res = await save({ name: name.trim(), active })
    if (res.ok) {
      onStored(res.data)
      return undefined
    }
    if (res.response?.status === 412) return t('locations.resourceConflict')
    if (res.code === 'forbidden') return t('locations.forbidden')
    if (res.code === 'invalid_location' || res.code === 'invalid_request') return t('locations.invalidName')
    return t('locations.resourceSaveFailed')
  })
  return (
    <form onSubmit={onSubmit} noValidate aria-label={item ? t('locations.editNamed', { name: item.name }) : t(l.add)}>
      <Field label={t(l.name)} maxLength={200} value={name} onChange={(e) => setName(e.target.value)} />
      <label className="check">
        <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} />
        <span>{t('locations.active')}</span>
      </label>
      <Alert message={error} />
      <div className="actions">
        <Submit busy={busy} label={t(l.save)} busyLabel={t('common.saving')} />
        <button type="button" onClick={onCancel} disabled={busy}>
          {t('locations.cancel')}
        </button>
      </div>
    </form>
  )
}
