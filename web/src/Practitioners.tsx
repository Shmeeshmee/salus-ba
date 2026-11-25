import { useEffect, useState } from 'react'
import { useIntl } from 'react-intl'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useFormat, useT } from './i18n/useT'
import { useLoad } from './load'
import { loadStaff, type Member } from './staff'
import { Alert, Confirm, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Invite = components['schemas']['Invite']
type Specialty = components['schemas']['Specialty']
type Load<V> = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; value: V }
type T = ReturnType<typeof useT>

// useSpecialtyName names a specialty from the catalog by its SNOMED CT code (SPEC §3 "Server text"), and by the
// API's text only for a code the catalog does not have.
function useSpecialtyName() {
  const intl = useIntl()
  return (code: string, text: string) => {
    const id = `common.specialty.${code}`
    return id in intl.messages ? intl.formatMessage({ id }) : text
  }
}

// Practitioners lists the clinic's practitioners (PractitionerRole search with the Practitioners included)
// with their specialty and, for an admin, the specialty editor, the open invites and the invite form.
export function Practitioners({ api, fhir, account }: { api: ApiClient; fhir: FhirClient; account: Account }) {
  const t = useT()
  const organizationId = account.organizationId ?? ''
  const admin = account.role === 'admin'
  const { state: members } = useLoad(() => loadStaff(fhir, organizationId), [fhir, organizationId])
  const { state: specialties } = useLoad(async () => {
    if (!admin) return []
    const res = await attempt(api.GET('/specialties'))
    if (!res.ok) throw new Error('specialties')
    return res.data.specialties
  }, [api, admin])

  return (
    <>
      <section className="panel">
        <h1>{t('admin.practitioners')}</h1>
        {members.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {members.name === 'failed' && <Alert message={t('admin.practitionersLoadFailed')} />}
        {members.name === 'loaded' && (
          <MemberList api={api} members={members.value} specialties={specialties.name === 'loaded' ? specialties.value : []} />
        )}
      </section>
      {admin && <Invites api={api} />}
    </>
  )
}

function MemberList({ api, members, specialties }: { api: ApiClient; members: Member[]; specialties: Specialty[] }) {
  const t = useT()
  const specialtyName = useSpecialtyName()
  if (members.length === 0) return <p className="muted">{t('common.noPractitioners')}</p>
  return (
    <ul className="rows" aria-label={t('admin.practitioners')}>
      {members.map((m) => (
        <li key={m.roleId}>
          <span className="stack">
            <span>{m.name}</span>
            {!m.active && <span className="tag">{t('admin.inactive')}</span>}
          </span>
          {specialties.length > 0 ? (
            <SpecialtyEditor api={api} member={m} specialties={specialties} />
          ) : (
            m.specialty && <span className="muted small">{specialtyName(m.specialty.code, m.specialty.text)}</span>
          )}
        </li>
      ))}
    </ul>
  )
}

// SpecialtyEditor saves a role's specialty as soon as the admin picks one, with If-Match on the role's
// version; a refused save puts the previous choice back.
function SpecialtyEditor({ api, member, specialties }: { api: ApiClient; member: Member; specialties: Specialty[] }) {
  const t = useT()
  const specialtyName = useSpecialtyName()
  const [code, setCode] = useState(member.specialty?.code ?? '')
  const [version, setVersion] = useState(member.version)
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<{ ok: boolean; message: string }>()

  async function choose(next: string) {
    const previous = code
    setCode(next)
    setBusy(true)
    setResult(undefined)
    const res = await attempt(
      api.PUT('/practitioner-roles/{id}/specialty', {
        params: { path: { id: member.roleId }, header: { 'If-Match': `W/"${version}"` } },
        body: { code: next },
      }),
    )
    setBusy(false)
    if (res.ok) {
      setVersion(String(res.data.versionId))
      setResult({ ok: true, message: t('admin.specialtySaved') })
      return
    }
    setCode(previous)
    setResult({ ok: false, message: specialtyMessage(t, res.code) })
  }

  return (
    <span className="specialty">
      <select aria-label={t('admin.specialtyOf', { name: member.name })} value={code} disabled={busy} onChange={(e) => choose(e.target.value)}>
        {code === '' && (
          <option value="" disabled>
            {t('admin.chooseSpecialty')}
          </option>
        )}
        {specialties.map((s) => (
          <option key={s.code} value={s.code}>
            {specialtyName(s.code, s.text)}
          </option>
        ))}
      </select>
      {result?.ok && <Notice message={result.message} />}
      {result && !result.ok && <Alert message={result.message} />}
    </span>
  )
}

function specialtyMessage(t: T, code?: string): string {
  if (code === 'version_conflict') return t('admin.specialtyConflict')
  if (code === 'invalid_specialty') return t('admin.invalidSpecialty')
  if (code === 'forbidden') return t('admin.specialtyForbidden')
  return t('common.somethingWrong')
}

function Invites({ api }: { api: ApiClient }) {
  const t = useT()
  const [invites, setInvites] = useState<Load<Invite[]>>({ name: 'loading' })

  useEffect(() => {
    let current = true
    attempt(api.GET('/invites')).then((res) => {
      if (current) setInvites(res.ok ? { name: 'loaded', value: res.data.invites } : { name: 'failed' })
    })
    return () => {
      current = false
    }
  }, [api])

  function added(invite: Invite) {
    setInvites((prev) => {
      const others = prev.name === 'loaded' ? prev.value.filter((i) => i.email !== invite.email) : []
      return { name: 'loaded', value: [...others, invite] }
    })
  }

  function removed(id: string) {
    setInvites((prev) => (prev.name === 'loaded' ? { name: 'loaded', value: prev.value.filter((i) => i.id !== id) } : prev))
  }

  return (
    <>
      <section className="panel">
        <h2>{t('admin.openInvites')}</h2>
        {invites.name === 'loading' && <p className="muted">{t('common.loading')}</p>}
        {invites.name === 'failed' && <Alert message={t('admin.invitesLoadFailed')} />}
        {invites.name === 'loaded' && <InviteList api={api} invites={invites.value} onGone={removed} />}
      </section>
      <section className="panel">
        <h2>{t('admin.inviteSomeone')}</h2>
        <InviteForm api={api} onInvited={added} />
      </section>
    </>
  )
}

function InviteList({ api, invites, onGone }: { api: ApiClient; invites: Invite[]; onGone: (id: string) => void }) {
  const t = useT()
  const [left, setLeft] = useState<Left>()
  const gone = (id: string, why: Left) => {
    setLeft(why)
    onGone(id)
  }
  return (
    <>
      {left === 'withdrawn' && <Notice message={t('admin.inviteWithdrawn')} />}
      {left === 'gone' && <Alert message={t('admin.inviteGone')} />}
      {invites.length === 0 ? (
        <p className="muted">{t('admin.noInvites')}</p>
      ) : (
        <ul className="rows" aria-label={t('admin.openInvites')}>
          {invites.map((i) => (
            <InviteRow key={i.id} api={api} invite={i} onGone={gone} />
          ))}
        </ul>
      )}
    </>
  )
}

// InviteRow is an open invite with "Withdraw" (wave 6, U7): asked first, then DELETE /invites/{id}. An invite that
// is no longer open (404: accepted, replaced or expired since the list was read) leaves the list too, with a line
// saying so.
type Left = 'withdrawn' | 'gone'

function InviteRow({ api, invite, onGone }: { api: ApiClient; invite: Invite; onGone: (id: string, why: Left) => void }) {
  const t = useT()
  const f = useFormat()
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function withdraw() {
    setConfirming(false)
    setBusy(true)
    setError(undefined)
    const res = await withdrawInvite(api, invite.id)
    setBusy(false)
    if (res.ok) onGone(invite.id, 'withdrawn')
    else if (res.response?.status === 404) onGone(invite.id, 'gone')
    else setError(t('admin.withdrawFailed'))
  }

  return (
    <li>
      <span className="stack">
        <span>{invite.email}</span>
        <span className="muted small">{inviteLine(t, invite, f.dateTime(new Date(invite.expiresAt), browserZone))}</span>
      </span>
      <span className="actions">
        <button type="button" disabled={busy} aria-label={t('admin.withdrawInviteFor', { email: invite.email })} onClick={() => setConfirming(true)}>
          {t('admin.withdrawInvite')}
        </button>
      </span>
      <Alert message={error} />
      {confirming && (
        <Confirm
          question={t('admin.withdrawQuestion', { email: invite.email })}
          lines={[t('admin.withdrawEffect')]}
          confirm={t('admin.withdrawInvite')}
          cancel={t('admin.keepInvite')}
          danger
          busy={busy}
          onConfirm={() => void withdraw()}
          onCancel={() => setConfirming(false)}
        />
      )}
    </li>
  )
}

// withdrawInvite calls DELETE /invites/{id}, BG1's route of wave 6, through the same client (and its CSRF header).
// The generated client gains the route when the wave is integrated; until then the call is typed here by hand.
function withdrawInvite(api: ApiClient, id: string) {
  type Delete = (path: string, init: { params: { path: { id: string } } }) => Promise<{ data?: undefined; error?: unknown; response: Response }>
  const del = api.DELETE as unknown as Delete
  return attempt(del('/invites/{id}', { params: { path: { id } } }))
}

// An invite's expiry is shown in the admin's own time zone, as the browser keeps it.
const browserZone = Intl.DateTimeFormat().resolvedOptions().timeZone

function inviteLine(t: T, invite: Invite, date: string): string {
  if (invite.role === 'admin') return t('admin.inviteAdmin', { date })
  if (invite.role === 'reception') return t('admin.inviteReception', { date })
  return t('admin.invitePractitioner', { name: [invite.givenName, invite.familyName].join(' '), date })
}

type Role = Invite['role']

function InviteForm({ api, onInvited }: { api: ApiClient; onInvited: (invite: Invite) => void }) {
  const t = useT()
  const [role, setRole] = useState<Role>('practitioner')
  const [email, setEmail] = useState('')
  const [givenName, setGivenName] = useState('')
  const [familyName, setFamilyName] = useState('')
  const [sent, setSent] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSent(false)
    const body = role === 'practitioner' ? { email, role, givenName, familyName } : { email, role } // an admin and a receptionist by e-mail alone
    const res = await attempt(api.POST('/invites', { body }))
    if (res.ok) {
      onInvited(res.data)
      setEmail('')
      setGivenName('')
      setFamilyName('')
      setSent(true)
      return undefined
    }
    return inviteMessage(t, res.code)
  })

  return (
    <form onSubmit={onSubmit} noValidate>
      <fieldset className="choice">
        <legend>{t('admin.role')}</legend>
        <RoleOption value="practitioner" label={t('common.practitioner')} role={role} onChange={setRole} />
        <RoleOption value="admin" label={t('admin.admin')} role={role} onChange={setRole} />
        <RoleOption value="reception" label={t('admin.reception')} role={role} onChange={setRole} />
      </fieldset>
      <Field label={t('common.email')} type="email" autoComplete="off" maxLength={254} value={email} onChange={(e) => setEmail(e.target.value)} required />
      {role === 'practitioner' && (
        <div className="pair">
          <Field label={t('admin.givenName')} autoComplete="off" maxLength={200} value={givenName} onChange={(e) => setGivenName(e.target.value)} required />
          <Field label={t('admin.familyName')} autoComplete="off" maxLength={200} value={familyName} onChange={(e) => setFamilyName(e.target.value)} required />
        </div>
      )}
      <Alert message={error} />
      {sent && <Notice message={t('admin.inviteSent')} />}
      <Submit busy={busy} label={t('admin.sendInvite')} busyLabel={t('admin.sending')} />
    </form>
  )
}

function RoleOption({ value, label, role, onChange }: { value: Role; label: string; role: Role; onChange: (r: Role) => void }) {
  return (
    <label>
      <input type="radio" name="role" value={value} checked={role === value} onChange={() => onChange(value)} />
      {label}
    </label>
  )
}

function inviteMessage(t: T, code?: string): string {
  if (code === 'invalid_email') return t('admin.invalidEmail')
  if (code === 'invalid_name') return t('admin.invalidName')
  if (code === 'forbidden') return t('admin.inviteForbidden')
  if (code === 'invite_limit') return t('admin.inviteLimit')
  return t('common.somethingWrong')
}
