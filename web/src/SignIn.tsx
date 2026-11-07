import { useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { CodeForm, Enrolment } from './Enrolment'
import { useT } from './i18n/useT'
import { signInLanguages } from './language'
import { Alert, AuthCard, Field, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type LoginResponse = components['schemas']['LoginResponse']

type Step =
  | { name: 'password'; timedOut?: boolean }
  | { name: 'verify'; challenge: string }
  | { name: 'enrol'; challenge: string; secret: string; uri: string }
  | { name: 'done' }

type T = ReturnType<typeof useT>

// tooManyAttempts says how long to wait from the Retry-After seconds, when it gives a time.
function tooManyAttempts(t: T, retryAfter: string | null): string {
  const seconds = Number(retryAfter)
  if (!Number.isFinite(seconds) || seconds <= 0) return t('auth.tooManyAttempts')
  return t('auth.tooManyAttemptsMinutes', { minutes: Math.ceil(seconds / 60) })
}

// SignIn is the password step, then the code (verify) or, for staff without MFA, the enrolment with the QR
// code. The session arrives as the salus_session cookie.
// The sign-in sends the language matched from the browser, and the one picked on this screen (signInLanguages).
export function SignIn({ api, onSignedIn }: { api: ApiClient; onSignedIn: () => void }) {
  const t = useT()
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
    return t('common.somethingWrong')
  }

  function restart(): undefined {
    setStep({ name: 'password', timedOut: true })
    return undefined
  }

  async function login(email: string, password: string): Promise<string | undefined> {
    const res = await attempt(api.POST('/auth/login', { body: { email, password, ...signInLanguages() } }))
    if (res.ok) return next(res.data)
    if (res.code === 'wrong_credentials') return t('auth.wrongCredentials')
    if (res.code === 'invalid_request') return t('auth.enterEmailPassword')
    if (res.code === 'rate_limited') return tooManyAttempts(t, res.response?.headers.get('Retry-After') ?? null)
    return t('common.somethingWrong')
  }

  async function code(path: '/auth/mfa/verify' | '/auth/mfa/confirm', challenge: string, value: string) {
    const res = await attempt(api.POST(path, { body: { challenge, code: value } }))
    if (res.ok && res.data?.next === 'done') return next(res.data)
    if (!res.ok && res.code === 'wrong_code') return t('auth.wrongCode')
    if (!res.ok && res.code === 'invalid_challenge') return restart()
    return t('common.somethingWrong')
  }

  switch (step.name) {
    case 'password':
      return (
        <AuthCard title={t('auth.signInTitle')}>
          <Notice message={step.timedOut ? t('auth.timedOut') : undefined} />
          <PasswordForm submit={login} />
        </AuthCard>
      )
    case 'verify':
      return (
        <AuthCard title={t('auth.codeTitle')}>
          <p>{t('auth.codeIntro')}</p>
          <CodeForm label={t('auth.continue')} busyLabel={t('auth.checking')} submit={(c) => code('/auth/mfa/verify', step.challenge, c)} />
        </AuthCard>
      )
    case 'enrol':
      return (
        <AuthCard title={t('auth.enrolTitle')}>
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
  const t = useT()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const { busy, error, onSubmit } = useSubmit(() => submit(email, password))
  return (
    <form onSubmit={onSubmit} noValidate>
      <Field label={t('common.email')} type="email" autoComplete="username" value={email} onChange={(e) => setEmail(e.target.value)} required />
      <Field
        label={t('auth.password')}
        type="password"
        autoComplete="current-password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
      />
      <Alert message={error} />
      <Submit busy={busy} label={t('auth.signIn')} busyLabel={t('auth.signingIn')} />
    </form>
  )
}
