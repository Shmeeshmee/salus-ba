import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import { Enrolment } from './Enrolment'
import { somethingWrong, wrongCode } from './messages'
import { Alert, AuthCard, Field, Link, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Step =
  | { name: 'password' }
  | { name: 'enrol'; challenge: string; secret: string; uri: string }
  | { name: 'timedOut' }
  | { name: 'done' }

const passwordLength = 'Use 12 to 128 characters.'

// AcceptInvite sets the password for an invite, then enrols the authenticator app; the confirmed first code
// opens the session.
export function AcceptInvite({ api, onDone }: { api: ApiClient; onDone: () => void }) {
  // The token comes from the link's fragment; it is then removed from the address bar and the history entry,
  // so it is not left behind in the browser.
  const [token] = useState(() => location.hash.slice(1))
  useEffect(() => {
    if (location.hash) history.replaceState(null, '', location.pathname)
  }, [])
  const [step, setStep] = useState<Step>({ name: 'password' })

  if (!token) {
    return (
      <AuthCard title="Accept your invite">
        <Alert message="This link is incomplete. Open the link in your invite e-mail again." />
      </AuthCard>
    )
  }

  async function accept(password: string): Promise<string | undefined> {
    const res = await attempt(api.POST('/invites/accept', { body: { token, password } }))
    if (res.ok) {
      setStep({ name: 'enrol', ...res.data })
      return undefined
    }
    if (res.code === 'invalid_invite') return 'This invite no longer works. Ask your clinic admin for a new one.'
    if (res.code === 'email_taken') return 'This e-mail address already has a Salus account. Ask your clinic admin for help.'
    if (res.code === 'invalid_password') return passwordLength
    return somethingWrong
  }

  async function confirm(challenge: string, code: string): Promise<string | undefined> {
    const res = await attempt(api.POST('/auth/mfa/confirm', { body: { challenge, code } }))
    if (res.ok && res.data?.next === 'done') {
      setStep({ name: 'done' })
      onDone()
      return undefined
    }
    if (!res.ok && res.code === 'wrong_code') return wrongCode
    if (!res.ok && res.code === 'invalid_challenge') {
      setStep({ name: 'timedOut' })
      return undefined
    }
    return somethingWrong
  }

  switch (step.name) {
    case 'password':
      return (
        <AuthCard title="Accept your invite">
          <p>Choose a password for your Salus account.</p>
          <NewPasswordForm submit={accept} />
        </AuthCard>
      )
    case 'enrol':
      return (
        <AuthCard title="Set up your authenticator app">
          <Enrolment secret={step.secret} uri={step.uri} confirm={(code) => confirm(step.challenge, code)} />
        </AuthCard>
      )
    case 'timedOut':
      return (
        <AuthCard title="Set up your authenticator app">
          <Notice message="The set-up timed out. Your account exists: sign in with your new password to finish it." />
          <Link to="/sign-in" className="button primary">
            Sign in
          </Link>
        </AuthCard>
      )
    case 'done':
      return null
  }
}

function NewPasswordForm({ submit }: { submit: (password: string) => Promise<string | undefined> }) {
  const [password, setPassword] = useState('')
  const [repeat, setRepeat] = useState('')
  const { busy, error, onSubmit } = useSubmit(async () => {
    const length = [...password].length
    if (length < 12 || length > 128) return passwordLength
    if (password !== repeat) return 'The passwords do not match.'
    return submit(password)
  })
  return (
    <form onSubmit={onSubmit} noValidate>
      <Field
        label="New password"
        type="password"
        autoComplete="new-password"
        hint="12 to 128 characters."
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
      />
      <Field
        label="Repeat password"
        type="password"
        autoComplete="new-password"
        value={repeat}
        onChange={(e) => setRepeat(e.target.value)}
        required
      />
      <Alert message={error} />
      <Submit busy={busy} label="Accept invite" busyLabel="Accepting" />
    </form>
  )
}
