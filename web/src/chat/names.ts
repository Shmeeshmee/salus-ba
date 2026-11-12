import { useEffect, useState } from 'react'
import type { useT } from '../i18n/useT'
import type { Member } from '../staff'
import { ChatError, reasons, type ChatMessage, type Conversation, type ConversationParticipant } from './chatApi'

// names.ts turns what the chat API gives (opaque ids and codes) into what the screens say, through the catalog:
// a doctor by the clinic's staff list, the patient by their name when the app may read it, a system message from
// its code, an error from the API's code. Nothing is taken from a message's args as text.

type T = ReturnType<typeof useT>

// PatientName reads a patient's display name; undefined when it cannot be read.
export type PatientName = (patientId: string) => Promise<string | undefined>

// usePatientName is the patient's name once read, or undefined.
export function usePatientName(patientName: PatientName, patientId: string | undefined): string | undefined {
  const [name, setName] = useState<{ id: string; name?: string }>()
  useEffect(() => {
    if (!patientId) return
    let current = true
    patientName(patientId).then(
      (n) => current && setName({ id: patientId, name: n }),
      () => undefined,
    )
    return () => {
      current = false
    }
  }, [patientName, patientId])
  return name && name.id === patientId ? name.name : undefined
}

// doctorName is the clinic's name of the practitioner role, or "A doctor" for one the clinic's list has not.
export function doctorName(staff: Member[], roleId: string | undefined, t: T): string {
  return staff.find((m) => m.roleId === roleId)?.name ?? t('chat.aDoctor')
}

export function participantName(p: ConversationParticipant, staff: Member[], patient: string | undefined, t: T): string {
  return p.role === 'patient' ? (patient ?? t('chat.patient')) : doctorName(staff, p.practitioner_role_id, t)
}

// senderName is who wrote a message: by its participant while they take part, else by the sender's role.
export function senderName(m: ChatMessage, c: Conversation, staff: Member[], patient: string | undefined, t: T): string {
  const p = c.participants?.find((x) => x.id === m.sender_participant_id)
  if (p) return participantName(p, staff, patient, t)
  return m.sender_role === 'patient' ? (patient ?? t('chat.patient')) : t('chat.aDoctor')
}

const systemCodes = ['participant_added', 'participant_invited', 'participant_removed', 'consent_granted', 'consent_declined']

// systemText is a system message as two sentences from the catalog: what happened (to the doctor by name when the
// clinic's list has them, else to "a doctor"), and the reason when it has one.
export function systemText(m: ChatMessage, staff: Member[], t: T): { text: string; reason?: string } {
  const code = m.system?.code ?? ''
  const args = m.system?.args ?? {}
  if (!systemCodes.includes(code)) return { text: t('chat.system.other') }
  const doctor = staff.find((m) => m.roleId === args.practitioner_role_id)
  const text = doctor ? t(`chat.system.${code}`, { name: doctor.name }) : t(`chat.system.${code}.anonymous`)
  const reason = (reasons as readonly string[]).includes(args.reason) ? t('chat.system.reason', { reason: t(`chat.reason.${args.reason}`) }) : undefined
  return { text, reason }
}

const explained = ['forbidden', 'consent_required', 'not_found', 'rate_limited', 'too_large', 'unsupported_type', 'too_many_files']

// errorText is what a refused call says: the API's code where the catalog explains it, the action's own message
// (fallback, from the catalog) otherwise.
export function errorText(err: unknown, fallback: string, t: T): string {
  if (err instanceof ChatError && explained.includes(err.code)) return t(`chat.error.${err.code}`)
  return fallback
}
