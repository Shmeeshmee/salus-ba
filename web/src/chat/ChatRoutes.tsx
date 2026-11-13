import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Account, ApiClient } from '../api/client'
import { nameOf, type FhirClient, type Patient } from '../api/fhir'
import { loadStaff, loadTimeZone, type Member } from '../staff'
import { chatClient } from './chatClient'
import { ChatView } from './ChatView'
import { Conversations } from './Conversations'

const chatPath = /^\/chats\/([^/]+)$/

// ChatRoutes are the chat's screens for staff: /chats and /chats/{id}, on the real chat client built once. The
// clinic's doctors (its PractitionerRole search) name the doctors of a conversation and are the ones Add a doctor
// offers; times are in the clinic's zone; a patient's name is read once per patient (FHIR Patient read).
export function ChatRoutes({ api, fhir, account, path }: { api: ApiClient; fhir: FhirClient; account: Account; path: string }) {
  const organizationId = account.organizationId ?? ''
  const chat = useMemo(() => chatClient({ api }), [api])
  const [staff, setStaff] = useState<Member[]>([])
  const [zone, setZone] = useState(() => Intl.DateTimeFormat().resolvedOptions().timeZone)

  useEffect(() => {
    let current = true
    loadStaff(fhir, organizationId).then((s) => current && setStaff(s), () => undefined) // without it doctors show as "A doctor"
    loadTimeZone(fhir, organizationId).then((z) => current && setZone(z), () => undefined) // the browser's until then
    return () => {
      current = false
    }
  }, [fhir, organizationId])

  const names = useRef(new Map<string, Promise<string | undefined>>())
  const patientName = useCallback(
    (id: string) => {
      let name = names.current.get(id)
      if (!name) {
        name = fhir.get<Patient>(`Patient/${encodeURIComponent(id)}`).then(
          (p) => nameOf(p.name?.[0]) || undefined,
          () => undefined, // not at full access: the screens say "Patient"
        )
        names.current.set(id, name)
      }
      return name
    },
    [fhir],
  )

  const open = chatPath.exec(path)
  if (open) {
    const id = decodeURIComponent(open[1])
    return <ChatView key={id} chat={chat} id={id} staff={staff} practitionerId={account.practitionerId} organizationId={organizationId} zone={zone} patientName={patientName} />
  }
  return <Conversations chat={chat} zone={zone} patientName={patientName} />
}
