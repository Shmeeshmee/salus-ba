import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { CodeForm, Enrolment } from './Enrolment'
import { somethingWrong, tooManyAttempts, wrongCode } from './messages'
import { Alert, AuthCard, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type LoginResponse = components['schemas']['LoginResponse']

type Step =
  | { name: 'password'; notice?: string }
  | { name: 'verify'; challenge: string }
  | { name: 'enrol'; challenge: string; secret: string; uri: string }
  | { name: 'done' }

const timedOut = 'Your sign-in timed out. Sign in again.'

// SignIn is the password step, then the code (verify) or, for staff without MFA, the enrolment with the QR
// code. The session arrives as the salus_session cookie.
export function SignIn({ api, onSignedIn }: { api: ApiClient; onSignedIn: () => void }) {
  const [step, setStep] = useState<Step>({ name: 'password' })

  async function next(res: LoginResponse): Promise<string | undefined> {
    if (res.next === 'done') {
      setStep({ name: 'done' })
      onSignedIn()
      return undefined
    }
    const challenge = res.challenge ?? ''
    if (res.next === 'verify') {
      setStep({ name: 'verify', challenge })
      return undefined
    }
    const enrol = await attempt(api.POST('/auth/mfa/enrol', { body: { challenge } }))
    if (enrol.ok) {
      setStep({ name: 'enrol', challenge, ...enrol.data })
      return undefined
    }
    if (enrol.code === 'invalid_challenge') return restart()
    return somethingWrong
  }

  function restart(): undefined {
    setStep({ name: 'password', notice: timedOut })
    return undefined
  }

  async function login(email: string, password: string): Promise<string | undefined> {
    const res = await attempt(api.POST('/auth/login', { body: { email, password } }))
    if (res.ok) return next(res.data)
    if (res.code === 'wrong_credentials') return 'The e-mail or password is wrong.'
    if (res.code === 'invalid_request') return 'Enter your e-mail and password.'
    if (res.code === 'rate_limited') return tooManyAttempts(res.response?.headers.get('Retry-After') ?? null)
    return somethingWrong
  }

  async function code(path: '/auth/mfa/verify' | '/auth/mfa/confirm', challenge: string, value: string) {
    const res = await attempt(api.POST(path, { body: { challenge, code: value } }))
    if (res.ok && res.data?.next === 'done') return next(res.data)
    if (!res.ok && res.code === 'wrong_code') return wrongCode
    if (!res.ok && res.code === 'invalid_challenge') return restart()
    return somethingWrong
  }

  switch (step.name) {
    case 'password':
      return (
        <AuthCard title="Sign in">
          <Notice message={step.notice} />
          <PasswordForm submit={login} />
        </AuthCard>
      )
    case 'verify':
      return (
        <AuthCard title="Enter your code">
          <p>Open your authenticator app and enter the 6-digit code for Salus.</p>
          <CodeForm label="Continue" busyLabel="Checking" submit={(c) => code('/auth/mfa/verify', step.challenge, c)} />
        </AuthCard>
      )
    case 'enrol':
      return (
        <AuthCard title="Set up your authenticator app">
          <Enrolment
            secret={step.secret}
            uri={step.uri}
            confirm={(c) => code('/auth/mfa/confirm', step.challenge, c)}
          />
        </AuthCard>
      )
    case 'done':
      return null
  }
}

function PasswordForm({ submit }: { submit: (email: string, password: string) => Promise<string | undefined> }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const { busy, error, onSubmit } = useSubmit(() => submit(email, password))
  return (
    <form onSubmit={onSubmit} noValidate>
      <Field label="E-mail" type="email" autoComplete="username" value={email} onChange={(e) => setEmail(e.target.value)} required />
      <Field
        label="Password"
        type="password"
        autoComplete="current-password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
      />
      <Alert message={error} />
      <Submit busy={busy} label="Sign in" busyLabel="Signing in" />
    </form>
  )
}
