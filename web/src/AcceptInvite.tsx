import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import { Enrolment } from './Enrolment'
import { useT } from './i18n/useT'
import { signInLanguages } from './language'
import { Alert, AuthCard, Field, Link, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

type Step =
  | { name: 'password' }
  | { name: 'enrol'; challenge: string; secret: string; uri: string }
  | { name: 'timedOut' }
  | { name: 'done' }

// AcceptInvite sets the password for an invite, then enrols the authenticator app; the confirmed first code
// opens the session. The acceptance sends the languages as a sign-in does (signInLanguages).
export function AcceptInvite({ api, onDone }: { api: ApiClient; onDone: () => void }) {
  const t = useT()
  // The token comes from the link's fragment; it is then removed from the address bar and the history entry,
  // so it is not left behind in the browser.
  const [token] = useState(() => location.hash.slice(1))
  useEffect(() => {
    if (location.hash) history.replaceState(null, '', location.pathname)
  }, [])
  const [step, setStep] = useState<Step>({ name: 'password' })

  if (!token) {
    return (
      <AuthCard title={t('auth.inviteTitle')}>
        <Alert message={t('auth.incompleteLink')} />
      </AuthCard>
    )
  }

  async function accept(password: string): Promise<string | undefined> {
    const res = await attempt(api.POST('/invites/accept', { body: { token, password, ...signInLanguages() } }))
    if (res.ok) {
      setStep({ name: 'enrol', ...res.data })
      return undefined
    }
    if (res.code === 'invalid_invite') return t('auth.invalidInvite')
    if (res.code === 'email_taken') return t('auth.emailTaken')
    if (res.code === 'invalid_password') return t('auth.passwordLength')
    return t('common.somethingWrong')
  }

  async function confirm(challenge: string, code: string): Promise<string | undefined> {
    const res = await attempt(api.POST('/auth/mfa/confirm', { body: { challenge, code } }))
    if (res.ok && res.data?.next === 'done') {
      setStep({ name: 'done' })
      onDone()
      return undefined
    }
    if (!res.ok && res.code === 'wrong_code') return t('auth.wrongCode')
    if (!res.ok && res.code === 'invalid_challenge') {
      setStep({ name: 'timedOut' })
      return undefined
    }
    return t('common.somethingWrong')
  }

  switch (step.name) {
    case 'password':
      return (
        <AuthCard title={t('auth.inviteTitle')}>
          <p>{t('auth.inviteIntro')}</p>
          <NewPasswordForm submit={accept} />
        </AuthCard>
      )
    case 'enrol':
      return (
        <AuthCard title={t('auth.enrolTitle')}>
          <Enrolment secret={step.secret} uri={step.uri} confirm={(code) => confirm(step.challenge, code)} />
        </AuthCard>
      )
    case 'timedOut':
      return (
        <AuthCard title={t('auth.enrolTitle')}>
          <Notice message={t('auth.setupTimedOut')} />
          <Link to="/sign-in" className="button primary">
            {t('auth.signIn')}
          </Link>
        </AuthCard>
      )
    case 'done':
      return null
  }
}

function NewPasswordForm({ submit }: { submit: (password: string) => Promise<string | undefined> }) {
  const t = useT()
  const [password, setPassword] = useState('')
  const [repeat, setRepeat] = useState('')
  const { busy, error, onSubmit } = useSubmit(async () => {
    const length = [...password].length
    if (length < 12 || length > 128) return t('auth.passwordLength')
    if (password !== repeat) return t('auth.passwordMismatch')
    return submit(password)
  })
  return (
    <form onSubmit={onSubmit} noValidate>
      <Field
        label={t('auth.newPassword')}
        type="password"
        autoComplete="new-password"
        hint={t('auth.newPasswordHint')}
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
      />
      <Field
        label={t('auth.repeatPassword')}
        type="password"
        autoComplete="new-password"
        value={repeat}
        onChange={(e) => setRepeat(e.target.value)}
        required
      />
      <Alert message={error} />
      <Submit busy={busy} label={t('auth.acceptInvite')} busyLabel={t('auth.accepting')} />
    </form>
  )
}
