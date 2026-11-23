import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { useT } from './i18n/useT'
import { Alert, Link } from './ui'

type Setup = components['schemas']['ClinicSetup']
type Step = keyof Setup
type State = components['schemas']['SetupStepState']

// The wizard's steps in order, each with its label and the screen where it is done (docs/clinic-cms, O1).
const steps: { step: Step; label: string; to: string }[] = [
  { step: 'details', label: 'signup.stepDetails', to: '/clinic' },
  { step: 'billing', label: 'signup.stepBilling', to: '/membership' },
  { step: 'hours', label: 'signup.stepHours', to: '/availability' },
  { step: 'practitioner', label: 'signup.stepPractitioner', to: '/practitioners' },
  { step: 'services', label: 'signup.stepServices', to: '/price-list' },
]

const stateLabel: Record<State, string> = { open: 'signup.stateOpen', done: 'signup.stateDone', skipped: 'signup.stateSkipped' }

// SetupWizard is the clinic admin's setup list: each step links to the screen that does it and can be marked done,
// skipped or reopened; the progress is the clinic's, so the admin resumes where they left off.
export function SetupWizard({ api }: { api: ApiClient }) {
  const t = useT()
  const [setup, setSetup] = useState<Setup>()
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    let current = true
    attempt(api.GET('/clinic/setup')).then((res) => {
      if (!current) return
      if (res.ok) setSetup(res.data)
      else setFailed(true)
    })
    return () => {
      current = false
    }
  }, [api])

  async function save(step: Step, state: State) {
    if (!setup) return
    setFailed(false)
    const res = await attempt(api.PUT('/clinic/setup', { body: { ...setup, [step]: state } }))
    if (res.ok) setSetup(res.data)
    else setFailed(true)
  }

  return (
    <section className="panel">
      <h1>{t('signup.setupTitle')}</h1>
      <p>{t('signup.setupIntro')}</p>
      <Alert message={failed ? t('common.somethingWrong') : undefined} />
      {setup && (
        <ol className="setup-steps">
          {steps.map(({ step, label, to }) => (
            <li key={step} className={`setup-step ${setup[step]}`}>
              <h2>{t(label)}</h2>
              <span className="tag">{t(stateLabel[setup[step]])}</span>
              <div className="actions">
                <Link to={to} className="button">
                  {t('signup.open')}
                </Link>
                {setup[step] === 'open' ? (
                  <>
                    <button type="button" className="primary" onClick={() => save(step, 'done')}>
                      {t('signup.markDone')}
                    </button>
                    <button type="button" className="quiet" onClick={() => save(step, 'skipped')}>
                      {t('signup.skip')}
                    </button>
                  </>
                ) : (
                  <button type="button" className="quiet" onClick={() => save(step, 'open')}>
                    {t('signup.reopen')}
                  </button>
                )}
              </div>
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}

// SetupBanner tells the admin, above every screen, that a step of the setup is still open; it shows nothing once
// every step is done or skipped, or when the progress cannot be read.
export function SetupBanner({ api }: { api: ApiClient }) {
  const t = useT()
  const [open, setOpen] = useState(false)
  useEffect(() => {
    let current = true
    attempt(api.GET('/clinic/setup')).then((res) => {
      if (current && res.ok) setOpen(Object.values(res.data).some((s) => s === 'open'))
    })
    return () => {
      current = false
    }
  }, [api])
  if (!open) return null
  return (
    <p className="panel notice-bar" role="status">
      {t('signup.banner')} <Link to="/setup">{t('signup.continueSetup')}</Link>
    </p>
  )
}
