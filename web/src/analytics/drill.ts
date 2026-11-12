import type { DiagnosisGroup, Dimension, Place } from './data'

// The drill-down of the analytics filters: what the diagnosis bars break down by, and the platform filter of a
// place.

export type Diagnosis = { group?: string; subgroup?: string }

// drillOf is the breakdown of the diagnoses under the selection: groups, a group's subgroups, then codes.
export function drillOf(value: Diagnosis, groups: DiagnosisGroup[]): 'group' | 'subgroup' | 'code' {
  if (!value.group) return 'group'
  if (!value.subgroup && groups.some((g) => g.parent === value.group)) return 'subgroup'
  return 'code'
}

// drillInto is the selection after opening a bar of the breakdown by.
export function drillInto(value: Diagnosis, by: 'group' | 'subgroup' | 'code', key: string): Diagnosis {
  if (by === 'group') return { group: key }
  if (by === 'subgroup') return { ...value, subgroup: key }
  return value
}

// Geo is the place chosen: place is the entity or a region, city one of the region's cities.
export type Geo = { place?: string; city?: string }

// geoFilter is the platform filter of a place: the deepest level chosen.
export function geoFilter(geo: Geo, places: Place[]): Partial<Record<Dimension, string>> {
  if (geo.city) return { city: geo.city }
  const kind = places.find((p) => p.id === geo.place)?.kind
  if (kind === 'entity') return { entity: geo.place }
  if (kind === 'region') return { region: geo.place }
  return {}
}

