import { useEffect, useMemo, useState } from 'react'
import { AcceptInvite } from './AcceptInvite'
import { AppointmentDetail } from './AppointmentDetail'
import { Availability } from './Availability'
import { Builder } from './Builder'
import { Calendar } from './Calendar'
import { CallScreen } from './CallScreen'
import { CallHistory } from './History'
import { consultSocket } from './live'
import { ClinicSettings } from './ClinicSettings'
import { Practitioners } from './Practitioners'
import { Questionnaires } from './Questionnaires'
import { SignIn } from './SignIn'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import { isStaffLanguage, type Language } from './i18n/languages'
import { useLanguage, useT } from './i18n/useT'
import { CalendarIcon, ClinicIcon, ClockIcon, ListIcon, Logo, PeopleIcon, ReviewIcon, SignOutIcon } from './icons'
import { LanguagePicker } from './LanguagePicker'
import { navigate, usePath } from './route'
import { Cheatsheet } from './terminology/Cheatsheet'
import { Review } from './terminology/Review'
import { Alert, Link, Notice } from './ui'

type Clients = { api: ApiClient; fhir: FhirClient }

// App picks the screen from the path: sign in and accept invite are open; everything else needs a session.
export default function App({ api, fhir }: Clients) {
  const path = usePath()
  if (path === '/sign-in') return <SignIn api={api} onSignedIn={() => navigate('/', true)} />
  if (path === '/invite') return <AcceptInvite api={api} onDone={() => navigate('/', true)} />
  return <SignedIn api={api} fhir={fhir} path={path} />
}

type Me = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; account: Account }

// A language saved from the header (shell L4), or a save that failed; shown below the header.
type LanguageSave = 'saved' | 'failed' | undefined

// staffChoice is the account's language when it is one the web app offers.
function staffChoice(account: Account): Language | null {
  return isStaffLanguage(account.language) ? account.language : null
}

function SignedIn({ api, fhir, path }: Clients & { path: string }) {
  const t = useT()
  const { choose } = useLanguage()
  const [me, setMe] = useState<Me>({ name: 'loading' })
  const [signOutFailed, setSignOutFailed] = useState(false)
  const [languageSave, setLanguageSave] = useState<LanguageSave>()

  // The account's language wins over the one kept on the device and the browser's (SPEC §3 "Picking and
  // saving"); without one, the app follows the browser.
  useEffect(() => {
    let current = true
    attempt(api.GET('/auth/me')).then((res) => {
      if (!current) return
      if (res.ok) {
        choose(staffChoice(res.data))
        setMe({ name: 'loaded', account: res.data })
      } else if (res.code === 'unauthenticated') navigate('/sign-in', true)
      else setMe({ name: 'failed' })
    })
    return () => {
      current = false
    }
    // choose is new with every language; the account's language is applied once per read of the account.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [api])

  async function signOut() {
    setSignOutFailed(false)
    const res = await attempt(api.POST('/auth/logout'))
    if (res.ok || res.code === 'unauthenticated') navigate('/sign-in', true)
    else setSignOutFailed(true)
  }

  // saveLanguage switches the UI and the picker at once and saves the choice on the account (PATCH /auth/me).
  async function saveLanguage(account: Account, language: Language | null) {
    choose(language)
    setMe({ name: 'loaded', account: { ...account, language: language ?? undefined } })
    setLanguageSave(undefined)
    const res = await attempt(api.PATCH('/auth/me', { body: { language } }))
    setLanguageSave(res.ok ? 'saved' : 'failed')
  }

  if (me.name === 'loading') return null
  if (me.name === 'failed') {
    return (
      <main className="auth">
        <Alert message={t('common.somethingWrong')} />
      </main>
    )
  }
  const account = me.account
  const staff = account.role !== 'patient'
  return (
    <>
      <header className="bar">
        <Logo />
        {staff && <Navigation admin={account.role === 'admin'} reviewer={account.termReviewer === true} />}
        {staff && <Cheatsheet api={api} />}
        {staff && <LanguagePicker value={staffChoice(account)} onChange={(language) => saveLanguage(account, language)} />}
        <button type="button" className="quiet" onClick={signOut}>
          <SignOutIcon />
          {t('shell.signOut')}
        </button>
      </header>
      <main className={path === '/' || callPath.test(path) || questionnairePath.test(path) ? 'page wide' : 'page'}>
        <Alert message={signOutFailed ? t('shell.signOutFailed') : undefined} />
        <Alert message={languageSave === 'failed' ? t('common.somethingWrong') : undefined} />
        <Notice message={languageSave === 'saved' ? t('shell.languageSaved') : undefined} />
        <Screen api={api} fhir={fhir} path={path} account={account} />
      </main>
    </>
  )
}

function Navigation({ admin, reviewer }: { admin: boolean; reviewer: boolean }) {
  const t = useT()
  return (
    <nav aria-label={t('shell.mainNav')}>
      <Link to="/">
        <CalendarIcon />
        {t('shell.calendar')}
      </Link>
      <Link to="/availability">
        <ClockIcon />
        {t('shell.workingHours')}
      </Link>
      <Link to="/practitioners">
        <PeopleIcon />
        {t('shell.practitioners')}
      </Link>
      <Link to="/questionnaires">
        <ListIcon />
        {t('shell.questionnaires')}
      </Link>
      {reviewer && (
        <Link to="/term-review">
          <ReviewIcon />
          {t('shell.termReview')}
        </Link>
      )}
      {admin && (
        <Link to="/clinic">
          <ClinicIcon />
          {t('shell.clinicSettings')}
        </Link>
      )}
    </nav>
  )
}

const appointmentPath = /^\/appointments\/([^/]+)$/
const callPath = /^\/appointments\/([^/]+)\/call$/
const questionnairePath = /^\/questionnaires\/([^/]+)$/

// Call is the call page with the consultation's live source and actions, built once for the page (live.ts).
function Call({ api, fhir, id }: Clients & { id: string }) {
  const consult = useMemo(() => consultSocket({ api, fhir }), [api, fhir])
  return <CallScreen api={api} id={id} consult={consult} history={<CallHistory api={api} fhir={fhir} appointmentId={id} />} />
}

function Screen({ api, fhir, path, account }: Clients & { path: string; account: Account }) {
  const t = useT()
  if (account.role === 'patient') {
    return <p className="panel">{t('shell.patientsUseApp')}</p>
  }
  const call = callPath.exec(path)
  if (call) return <Call api={api} fhir={fhir} id={decodeURIComponent(call[1])} />
  const appointment = appointmentPath.exec(path)
  if (appointment) return <AppointmentDetail api={api} fhir={fhir} account={account} id={decodeURIComponent(appointment[1])} />
  if (path === '/') return <Calendar fhir={fhir} account={account} />
  if (path === '/availability') return <Availability api={api} fhir={fhir} account={account} />
  if (path === '/practitioners') return <Practitioners api={api} fhir={fhir} account={account} />
  if (path === '/questionnaires') return <Questionnaires api={api} fhir={fhir} account={account} />
  const questionnaire = questionnairePath.exec(path)
  if (questionnaire) return <Builder key={questionnaire[1]} api={api} fhir={fhir} id={decodeURIComponent(questionnaire[1])} />
  if (path === '/term-review') return <Review api={api} reviewer={account.termReviewer === true} />
  if (path === '/clinic' && account.role === 'admin') {
    return <ClinicSettings fhir={fhir} organizationId={account.organizationId ?? ''} />
  }
  return <p className="panel">{t('shell.notFound')}</p>
}
