import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import { useLanguage, useT } from './i18n/useT'
import { Alert, AuthCard, Field, Link, Notice, Submit } from './ui'
import { useSubmit } from './useSubmit'

// Signup is the public sign-up of a clinic (docs/clinic-cms, O1): the clinic's name and its admin's e-mail. The
// answer is the same whether or not the address is known, so the page only says to check the e-mail.
export function Signup({ api }: { api: ApiClient }) {
  const t = useT()
  const { language } = useLanguage()
  const [email, setEmail] = useState('')
  const [clinicName, setClinicName] = useState('')
  const [sent, setSent] = useState(false)
  const { busy, error, onSubmit } = useSubmit(async () => {
    if (!email.trim() || !clinicName.trim()) return t('signup.enterBoth')
    const res = await attempt(api.POST('/signup', { body: { email: email.trim(), clinicName: clinicName.trim(), language } }))
    if (res.ok) {
      setSent(true)
      return undefined
    }
    if (res.code === 'invalid_signup' || res.code === 'invalid_request') return t('signup.invalid')
    if (res.code === 'rate_limited') return t('signup.tooMany')
    return t('common.somethingWrong')
  })
  if (sent) {
    return (
      <AuthCard title={t('signup.title')}>
        <Notice message={t('signup.checkEmail')} />
      </AuthCard>
    )
  }
  return (
    <AuthCard title={t('signup.title')}>
      <p>{t('signup.intro')}</p>
      <form onSubmit={onSubmit} noValidate>
        <Field label={t('signup.clinicName')} autoComplete="organization" maxLength={200} value={clinicName} onChange={(e) => setClinicName(e.target.value)} required />
        <Field label={t('signup.email')} type="email" autoComplete="email" maxLength={254} value={email} onChange={(e) => setEmail(e.target.value)} required />
        <Alert message={error} />
        <Submit busy={busy} label={t('signup.submit')} busyLabel={t('signup.submitting')} />
      </form>
      <p>
        <Link to="/sign-in">{t('signup.haveAccount')}</Link>
      </p>
    </AuthCard>
  )
}

type Verification = 'checking' | 'verified' | 'invalid' | 'failed'

// VerifySignup confirms the e-mail with the token of the link's fragment; the token is then removed from the address
// bar and the history entry. The clinic is created and its admin invited: the invite e-mail follows.
export function VerifySignup({ api }: { api: ApiClient }) {
  const t = useT()
  const [token] = useState(() => location.hash.slice(1))
  const [state, setState] = useState<Verification>('checking')
  useEffect(() => {
    if (location.hash) history.replaceState(null, '', location.pathname)
    if (!token) return
    let current = true
    attempt(api.POST('/signup/verify', { body: { token } })).then((res) => {
      if (!current) return
      if (res.ok) setState('verified')
      else setState(res.code === 'invalid_verification' || res.code === 'invalid_request' ? 'invalid' : 'failed')
    })
    return () => {
      current = false
    }
  }, [api, token])

  if (!token) {
    return (
      <AuthCard title={t('signup.verifyTitle')}>
        <Alert message={t('signup.incompleteLink')} />
      </AuthCard>
    )
  }
  return (
    <AuthCard title={t('signup.verifyTitle')}>
      <Notice message={state === 'verified' ? t('signup.verified') : undefined} />
      <Alert message={state === 'invalid' ? t('signup.invalidLink') : state === 'failed' ? t('common.somethingWrong') : undefined} />
      {state === 'invalid' && <Link to="/signup">{t('signup.submit')}</Link>}
    </AuthCard>
  )
}
