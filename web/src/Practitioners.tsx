import { useEffect, useState } from 'react'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useLoad } from './load'
import { somethingWrong } from './messages'
import { loadStaff, type Member } from './staff'
import { Alert, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Invite = components['schemas']['Invite']
type Specialty = components['schemas']['Specialty']
type Load<T> = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; value: T }

// Practitioners lists the clinic's practitioners (PractitionerRole search with the Practitioners included)
// with their specialty and, for an admin, the specialty editor, the open invites and the invite form.
export function Practitioners({ api, fhir, account }: { api: ApiClient; fhir: FhirClient; account: Account }) {
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
        <h1>Practitioners</h1>
        {members.name === 'loading' && <p className="muted">Loading</p>}
        {members.name === 'failed' && <Alert message="The practitioners could not be loaded. Reload the page to try again." />}
        {members.name === 'loaded' && (
          <MemberList api={api} members={members.value} specialties={specialties.name === 'loaded' ? specialties.value : []} />
        )}
      </section>
      {admin && <Invites api={api} />}
    </>
  )
}

function MemberList({ api, members, specialties }: { api: ApiClient; members: Member[]; specialties: Specialty[] }) {
  if (members.length === 0) return <p className="muted">No practitioners yet.</p>
  return (
    <ul className="rows" aria-label="Practitioners">
      {members.map((m) => (
        <li key={m.roleId}>
          <span className="stack">
            <span>{m.name}</span>
            {!m.active && <span className="tag">Inactive</span>}
          </span>
          {specialties.length > 0 ? (
            <SpecialtyEditor api={api} member={m} specialties={specialties} />
          ) : (
            m.specialty && <span className="muted small">{m.specialty.text}</span>
          )}
        </li>
      ))}
    </ul>
  )
}

// SpecialtyEditor saves a role's specialty as soon as the admin picks one, with If-Match on the role's
// version; a refused save puts the previous choice back.
function SpecialtyEditor({ api, member, specialties }: { api: ApiClient; member: Member; specialties: Specialty[] }) {
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
      setResult({ ok: true, message: 'Specialty saved.' })
      return
    }
    setCode(previous)
    setResult({ ok: false, message: specialtyMessage(res.code) })
  }

  return (
    <span className="specialty">
      <select aria-label={`Specialty of ${member.name}`} value={code} disabled={busy} onChange={(e) => choose(e.target.value)}>
        {code === '' && (
          <option value="" disabled>
            Choose a specialty
          </option>
        )}
        {specialties.map((s) => (
          <option key={s.code} value={s.code}>
            {s.text}
          </option>
        ))}
      </select>
      {result?.ok && <Notice message={result.message} />}
      {result && !result.ok && <Alert message={result.message} />}
    </span>
  )
}

function specialtyMessage(code?: string): string {
  if (code === 'version_conflict') return 'Someone else changed this practitioner. Reload the page to see the change.'
  if (code === 'invalid_specialty') return 'Choose a specialty from the list.'
  if (code === 'forbidden') return 'Only a clinic admin can set a specialty.'
  return somethingWrong
}

function Invites({ api }: { api: ApiClient }) {
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

  return (
    <>
      <section className="panel">
        <h2>Open invites</h2>
        {invites.name === 'loading' && <p className="muted">Loading</p>}
        {invites.name === 'failed' && <Alert message="The invites could not be loaded. Reload the page to try again." />}
        {invites.name === 'loaded' && <InviteList invites={invites.value} />}
      </section>
      <section className="panel">
        <h2>Invite someone</h2>
        <InviteForm api={api} onInvited={added} />
      </section>
    </>
  )
}

const expires = new Intl.DateTimeFormat('en-GB', { dateStyle: 'medium', timeStyle: 'short' })

function InviteList({ invites }: { invites: Invite[] }) {
  if (invites.length === 0) return <p className="muted">No open invites.</p>
  return (
    <ul className="rows" aria-label="Open invites">
      {invites.map((i) => (
        <li key={i.id}>
          <span className="stack">
            <span>{i.email}</span>
            <span className="muted small">
              {i.role === 'admin' ? 'Admin' : `Practitioner, ${[i.givenName, i.familyName].join(' ')}`}
              {` · Expires ${expires.format(new Date(i.expiresAt))}`}
            </span>
          </span>
        </li>
      ))}
    </ul>
  )
}

type Role = Invite['role']

function InviteForm({ api, onInvited }: { api: ApiClient; onInvited: (invite: Invite) => void }) {
  const [role, setRole] = useState<Role>('practitioner')
  const [email, setEmail] = useState('')
  const [givenName, setGivenName] = useState('')
  const [familyName, setFamilyName] = useState('')
  const [sent, setSent] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    setSent(false)
    const body = role === 'admin' ? { email, role } : { email, role, givenName, familyName }
    const res = await attempt(api.POST('/invites', { body }))
    if (res.ok) {
      onInvited(res.data)
      setEmail('')
      setGivenName('')
      setFamilyName('')
      setSent(true)
      return undefined
    }
    return inviteMessage(res.code)
  })

  return (
    <form onSubmit={onSubmit} noValidate>
      <fieldset className="choice">
        <legend>Role</legend>
        <RoleOption value="practitioner" label="Practitioner" role={role} onChange={setRole} />
        <RoleOption value="admin" label="Admin" role={role} onChange={setRole} />
      </fieldset>
      <Field label="E-mail" type="email" autoComplete="off" maxLength={254} value={email} onChange={(e) => setEmail(e.target.value)} required />
      {role === 'practitioner' && (
        <div className="pair">
          <Field label="Given name" autoComplete="off" maxLength={200} value={givenName} onChange={(e) => setGivenName(e.target.value)} required />
          <Field label="Family name" autoComplete="off" maxLength={200} value={familyName} onChange={(e) => setFamilyName(e.target.value)} required />
        </div>
      )}
      <Alert message={error} />
      {sent && <Notice message="Invite sent." />}
      <Submit busy={busy} label="Send invite" busyLabel="Sending" />
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

function inviteMessage(code?: string): string {
  if (code === 'invalid_email') return 'Enter a valid e-mail address.'
  if (code === 'invalid_name') return 'Enter the given and family name.'
  if (code === 'forbidden') return 'Only a clinic admin can send invites.'
  return somethingWrong
}
