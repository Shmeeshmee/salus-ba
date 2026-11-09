import { useState } from 'react'
import { useT } from './i18n/useT'
import { QRCode } from './QRCode'
import { Alert, Field, Submit } from './ui'
import { useSubmit } from './useSubmit'

// Enrolment shows a new TOTP secret as a QR code and as text, and asks for the first code. It is only
// mounted while enrolment is pending; the screens drop the secret as soon as the code is confirmed.
export function Enrolment({ secret, uri, confirm }: { secret: string; uri: string; confirm: (code: string) => Promise<string | undefined> }) {
  const t = useT()
  return (
    <>
      <p>{t('auth.enrolIntro')}</p>
      <QRCode value={uri} label={t('auth.qrLabel')} />
      <p className="hint">{t('auth.cannotScan')}</p>
      <code className="secret">{secret}</code>
      <CodeForm label={t('auth.confirm')} busyLabel={t('auth.confirming')} submit={confirm} />
    </>
  )
}

export function CodeForm({ label, busyLabel, submit }: { label: string; busyLabel: string; submit: (code: string) => Promise<string | undefined> }) {
  const t = useT()
  const [code, setCode] = useState('')
  const { busy, error, onSubmit } = useSubmit(() => submit(code.trim()))
  return (
    <form onSubmit={onSubmit} noValidate>
      <Field
        label={t('auth.code')}
        value={code}
        onChange={(e) => setCode(e.target.value)}
        inputMode="numeric"
        autoComplete="one-time-code"
        maxLength={6}
        required
      />
      <Alert message={error} />
      <Submit busy={busy} label={label} busyLabel={busyLabel} />
    </form>
  )
}
