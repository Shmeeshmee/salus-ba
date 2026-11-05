import { nameOf, settingsOf, type Bundle, type FhirClient, type Organization, type Practitioner } from './api/fhir'

// Member is one practitioner of the clinic with their role: what the lists, pickers and editors show.
export type Member = {
  roleId: string
  practitionerId: string
  name: string
  active: boolean
  specialty?: { code: string; text: string }
  version: string // the role's meta.versionId
}

// loadStaff reads the clinic's roles with their practitioners (PractitionerRole search, practitioners
// included).
export async function loadStaff(fhir: FhirClient, organizationId: string): Promise<Member[]> {
  const query = new URLSearchParams({ organization: organizationId, _include: 'PractitionerRole:practitioner' })
  return membersOf(await fhir.get<Bundle>(`PractitionerRole?${query}`))
}

function membersOf(bundle: Bundle): Member[] {
  const people = new Map<string, Practitioner>()
  for (const e of bundle.entry ?? []) if (e.resource.resourceType === 'Practitioner') people.set(`Practitioner/${e.resource.id}`, e.resource)
  const members: Member[] = []
  for (const e of bundle.entry ?? []) {
    if (e.resource.resourceType !== 'PractitionerRole') continue
    const r = e.resource
    const person = people.get(r.practitioner.reference)
    if (!person) continue
    const s = r.specialty?.[0]
    const code = s?.coding?.[0]?.code
    members.push({
      roleId: r.id,
      practitionerId: person.id,
      name: nameOf(person.name?.[0]),
      active: r.active && person.active,
      specialty: code ? { code, text: s?.text ?? s?.coding?.[0]?.display ?? code } : undefined,
      version: r.meta.versionId,
    })
  }
  return members
}

// loadTimeZone reads the clinic's time zone from its Organization.
export async function loadTimeZone(fhir: FhirClient, organizationId: string): Promise<string> {
  return settingsOf(await fhir.get<Organization>(`Organization/${encodeURIComponent(organizationId)}`)).timeZone
}
