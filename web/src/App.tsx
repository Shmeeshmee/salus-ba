import { useEffect, useState } from 'react'
import { AcceptInvite } from './AcceptInvite'
import { AppointmentDetail } from './AppointmentDetail'
import { Availability } from './Availability'
import { Calendar } from './Calendar'
import { ClinicSettings } from './ClinicSettings'
import { Practitioners } from './Practitioners'
import { SignIn } from './SignIn'
import type { Account, ApiClient } from './api/client'
import type { FhirClient } from './api/fhir'
import { attempt } from './api/result'
import { CalendarIcon, ClinicIcon, ClockIcon, Logo, PeopleIcon, SignOutIcon } from './icons'
import { somethingWrong } from './messages'
import { navigate, usePath } from './route'
import { Alert, Link } from './ui'

type Clients = { api: ApiClient; fhir: FhirClient }

// App picks the screen from the path: sign in and accept invite are open; everything else needs a session.
export default function App({ api, fhir }: Clients) {
  const path = usePath()
  if (path === '/sign-in') return <SignIn api={api} onSignedIn={() => navigate('/', true)} />
  if (path === '/invite') return <AcceptInvite api={api} onDone={() => navigate('/', true)} />
  return <SignedIn api={api} fhir={fhir} path={path} />
}

type Me = { name: 'loading' } | { name: 'failed' } | { name: 'loaded'; account: Account }

function SignedIn({ api, fhir, path }: Clients & { path: string }) {
  const [me, setMe] = useState<Me>({ name: 'loading' })
  const [signOutError, setSignOutError] = useState<string>()

  useEffect(() => {
    let current = true
    attempt(api.GET('/auth/me')).then((res) => {
      if (!current) return
      if (res.ok) setMe({ name: 'loaded', account: res.data })
      else if (res.code === 'unauthenticated') navigate('/sign-in', true)
      else setMe({ name: 'failed' })
    })
    return () => {
      current = false
    }
  }, [api])

  async function signOut() {
    setSignOutError(undefined)
    const res = await attempt(api.POST('/auth/logout'))
    if (res.ok || res.code === 'unauthenticated') navigate('/sign-in', true)
    else setSignOutError('Signing out failed. Try again.')
  }

  if (me.name === 'loading') return null
  if (me.name === 'failed') {
    return (
      <main className="auth">
        <Alert message={somethingWrong} />
      </main>
    )
  }
  const account = me.account
  const admin = account.role === 'admin'
  return (
    <>
      <header className="bar">
        <Logo />
        {account.role !== 'patient' && (
          <nav aria-label="Main">
            <Link to="/">
              <CalendarIcon />
              Calendar
            </Link>
            <Link to="/availability">
              <ClockIcon />
              Working hours
            </Link>
            <Link to="/practitioners">
              <PeopleIcon />
              Practitioners
            </Link>
            {admin && (
              <Link to="/clinic">
                <ClinicIcon />
                Clinic settings
              </Link>
            )}
          </nav>
        )}
        <button type="button" className="quiet" onClick={signOut}>
          <SignOutIcon />
          Sign out
        </button>
      </header>
      <main className={path === '/' ? 'page wide' : 'page'}>
        <Alert message={signOutError} />
        <Screen api={api} fhir={fhir} path={path} account={account} />
      </main>
    </>
  )
}

const appointmentPath = /^\/appointments\/([^/]+)$/

function Screen({ api, fhir, path, account }: Clients & { path: string; account: Account }) {
  if (account.role === 'patient') {
    return <p className="panel">The web app is for clinic staff. Patients use the Salus mobile app.</p>
  }
  const appointment = appointmentPath.exec(path)
  if (appointment) return <AppointmentDetail api={api} fhir={fhir} account={account} id={decodeURIComponent(appointment[1])} />
  if (path === '/') return <Calendar fhir={fhir} account={account} />
  if (path === '/availability') return <Availability api={api} fhir={fhir} account={account} />
  if (path === '/practitioners') return <Practitioners api={api} fhir={fhir} account={account} />
  if (path === '/clinic' && account.role === 'admin') {
    return <ClinicSettings fhir={fhir} organizationId={account.organizationId ?? ''} />
  }
  return <p className="panel">This page does not exist.</p>
}
