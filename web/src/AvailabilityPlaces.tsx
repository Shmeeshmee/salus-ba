import type { components } from './api/schema'
import { useT } from './i18n/useT'

type Location = components['schemas']['Location']
type Room = components['schemas']['Room']

// Places are the clinic's locations and each one's rooms, what an in-person row of working hours names (U2, U3 of
// docs/clinic-cms/WAVE6.md).
export type Places = { locations: Location[]; rooms: Record<string, Room[]> }

type Props = {
  places: Places
  n: number
  locationId?: string
  roomId?: string
  onChange: (next: { locationId?: string; roomId?: string }) => void
}

// PlaceSelects picks the location of an in-person row, among those in use and the row's own, and a room of it when
// the location has rooms; a new location clears the room. They sit on a line of their own below the row's times.
export function PlaceSelects({ places, n, locationId, roomId, onChange }: Props) {
  const t = useT()
  const locations = places.locations.filter((l) => l.active || l.id === locationId)
  if (locations.length === 0) return <p className="hint hours-place">{t('locations.noLocations')}</p>
  const rooms = (places.rooms[locationId ?? ''] ?? []).filter((r) => r.active || r.id === roomId)
  return (
    <span className="hours-place">
      <select aria-label={t('locations.hoursLocationN', { n })} value={locationId ?? ''} onChange={(e) => onChange({ locationId: e.target.value, roomId: undefined })}>
        {locations.map((l) => (
          <option key={l.id} value={l.id}>
            {l.name}
          </option>
        ))}
      </select>
      {rooms.length > 0 && (
        <select aria-label={t('locations.hoursRoomN', { n })} value={roomId ?? ''} onChange={(e) => onChange({ locationId, roomId: e.target.value || undefined })}>
          <option value="">{t('locations.noRoom')}</option>
          {rooms.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name}
            </option>
          ))}
        </select>
      )}
    </span>
  )
}
