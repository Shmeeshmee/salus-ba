import { expect, test, type Browser, type BrowserContext, type Locator, type Page } from '@playwright/test'
import { axe, dialog, findings, report, tabWalk, targets } from './a11y'
import { required, signIn } from './signin'

// Every staff screen and dialog of the web app for one role (E2E_A11Y_ROLE: admin, practitioner or reception), as the Go
// test TestAccessibility_EveryStaffScreenAndDialog signs it in (docs/clinic-cms/WAVE7.md RD3). At 1280 px: the axe
// engine (WCAG 2.2 A and AA), the keyboard walk of each screen and dialog (order, focus seen, focus back on the opener).
// At 390 px on a touch screen: the engine again and targets of at least 44 px. Every finding of the run is listed at
// the end.

type Role = 'admin' | 'practitioner' | 'reception'
type Screen = { name: string; path: string; open?: (page: Page) => Promise<void>; dialogs?: (page: Page, where: string, touch: boolean) => Promise<void> }

const visit = () => `/appointments/${required('E2E_A11Y_VISIT')}`
const button = (page: Page, name: string | RegExp) => page.getByRole('button', { name, exact: typeof name === 'string' }).first()
const named = (page: Page, name: string) => page.getByRole('dialog', { name })
const cancelIn = (name: string) => async (d: Locator) => {
  await d.getByRole('button', { name }).focus()
  await d.page().keyboard.press('Enter')
}

const deskDialogs = async (page: Page, where: string, touch: boolean) => {
  const checks = { tab: !touch, touch }
  if (process.env.E2E_A11Y_DESK_ROWS !== 'yes') return
  const patient = required('E2E_A11Y_DESK_PATIENT')
  await dialog(page, `${where}, walk-in`, button(page, 'Walk-in'), named(page, 'Walk-in'), undefined, checks)
  await dialog(page, `${where}, patients`, button(page, 'Patients'), named(page, 'Patients'), undefined, checks)
  const row = page.locator('li').filter({ hasText: patient })
  await dialog(page, `${where}, another receipt`, row.getByRole('button', { name: 'Issue another receipt' }), named(page, 'New receipt'), undefined, checks)
  const booked = page.locator('li').filter({ has: page.getByRole('button', { name: 'Move' }) }).first()
  await dialog(page, `${where}, move`, booked.getByRole('button', { name: 'Move' }), named(page, 'Move visit'), undefined, checks)
}

const clinical = (role: Role): Screen[] => [
  { name: 'calendar', path: '/' },
  { name: 'working hours', path: '/availability' },
  { name: 'practitioners', path: '/practitioners' },
  { name: 'questionnaires', path: '/questionnaires' },
  { name: 'waitlist', path: '/waitlist' },
  { name: 'analytics', path: '/analytics' },
  { name: 'term review', path: '/term-review' },
  {
    name: 'the visit',
    path: visit(),
    dialogs: async (page, where, touch) => {
      const checks = { tab: !touch, touch }
      await button(page, 'Show treatment plans').click()
      await dialog(page, `${where}, new treatment plan`, button(page, 'New treatment plan'), page.getByRole('dialog'), undefined, checks)
      await dialog(page, `${where}, book treatment plan`, button(page, 'Book'), named(page, 'Book treatment plan'), undefined, checks)
      await dialog(page, `${where}, cancel treatment plan`, button(page, 'Cancel plan'), page.getByRole('alertdialog'), cancelIn('Keep the plan'), checks)
      if (role === 'practitioner') await button(page, 'Add a lab result').click()
      if (role === 'admin') {
        await dialog(page, `${where}, receipt`, button(page, 'Issue receipt'), named(page, 'New receipt'), undefined, checks)
        await dialog(page, `${where}, quote`, button(page, 'Write a quote'), named(page, 'New quote'), undefined, checks)
      }
      if (role === 'practitioner') await button(page, 'Show history').click()
      await button(page, 'Show changes').click()
      await axe(page, `${where}, every section open`)
      if (touch) await targets(page, `${where}, every section open`)
    },
  },
  {
    name: 'the cheatsheet',
    path: '/',
    dialogs: async (page, where, touch) => {
      const opener = page.locator('header button.cheatsheet-open')
      await dialog(page, where, opener, page.locator('[role=dialog].cheatsheet'), undefined, { tab: !touch, touch })
    },
  },
]

function screens(role: Role): Screen[] {
  if (role === 'reception') return [{ name: 'desk', path: '/desk', dialogs: deskDialogs }]
  if (role === 'practitioner') {
    return [
      ...clinical(role),
      {
        name: 'messages',
        path: '/chats',
        open: async (page) => {
          await page.locator('main a[href^="/chats/"]').first().click()
          await expect(page).toHaveURL(/\/chats\/[^/]+$/)
        },
        dialogs: async (page, where, touch) => {
          await dialog(page, `${where}, add a doctor`, button(page, 'Add a doctor'), named(page, 'Add a doctor'), undefined, { tab: !touch, touch })
        },
      },
      {
        name: 'the questionnaire builder',
        path: '/questionnaires',
        open: async (page) => {
          await button(page, 'New questionnaire').click()
          await expect(page).toHaveURL(/\/questionnaires\/[^/]+$/)
        },
      },
    ]
  }
  return [
    { name: 'desk', path: '/desk', dialogs: deskDialogs },
    ...clinical(role),
    {
      name: 'receipts',
      path: '/receipts',
      dialogs: async (page, where, touch) => {
        await dialog(page, `${where}, new receipt`, button(page, 'New receipt'), named(page, 'New receipt'), undefined, { tab: !touch, touch })
      },
    },
    { name: 'receipts report', path: '/receipts/report' },
    { name: 'online practice', path: '/receipts/practice' },
    { name: 'quotes', path: '/quotes' },
    { name: 'data requests', path: '/data-requests' },
    {
      name: 'a data request',
      path: '/data-requests',
      open: async (page) => {
        await page.locator('main a[href^="/data-requests/"]').first().click()
        await expect(page).toHaveURL(/\/data-requests\/[^/]+$/)
      },
    },
    { name: 'membership', path: '/membership' },
    { name: 'clinic settings', path: '/clinic' },
    { name: 'price list', path: '/price-list' },
    { name: 'locations', path: '/locations' },
    { name: 'setup', path: '/setup' },
  ]
}

// ready waits until the screen has loaded: its heading is shown and nothing is loading.
async function ready(page: Page) {
  await expect(page.locator('main').getByRole('heading').first()).toBeVisible()
  await expect(page.locator('main').getByText(/^Loading/)).toHaveCount(0)
}

async function show(page: Page, s: Screen) {
  await page.goto(s.path)
  await ready(page)
  if (s.open) {
    await s.open(page)
    await ready(page)
  }
}

// errorsAnnounced: a sign-in with a wrong password says so in an alert.
async function errorsAnnounced(page: Page) {
  await page.goto('/sign-in')
  await axe(page, 'sign-in')
  await page.getByLabel('E-mail').fill(required('E2E_PRACTITIONER_EMAIL'))
  await page.getByLabel('Password').fill('not the password')
  await page.getByRole('button', { name: 'Sign in' }).click()
  const alert = page.getByRole('alert')
  await expect(alert).toBeVisible()
  if (!(await alert.textContent())?.trim()) findings.push('sign-in: the alert of a wrong password is empty')
  await tabWalk(page, 'sign-in, wrong password')
}

// checked runs one screen's checks; a step that cannot be done (a dialog that does not close, a control not found) is a
// finding too, and the walk goes on with the next screen.
async function checked(where: string, run: () => Promise<void>) {
  try {
    await run()
  } catch (err) {
    findings.push(`${where}: could not be checked: ${String(err instanceof Error ? err.message : err).split('\n')[0]}`)
  }
}

async function phoneContext(browser: Browser, desktop: BrowserContext) {
  return browser.newContext({
    baseURL: 'http://localhost:5173',
    viewport: { width: 390, height: 844 },
    hasTouch: true,
    isMobile: true,
    reducedMotion: 'reduce',
    storageState: await desktop.storageState(),
  })
}

test('every staff screen and dialog of the role passes the accessibility checks', async ({ browser }) => {
  test.setTimeout(11 * 60_000)
  const role = required('E2E_A11Y_ROLE') as Role
  const desktop = await browser.newContext({ baseURL: 'http://localhost:5173', viewport: { width: 1280, height: 900 }, reducedMotion: 'reduce' })
  const page = await desktop.newPage()
  if (role === 'admin') await errorsAnnounced(page)
  await signIn(page)
  const phone = await (await phoneContext(browser, desktop)).newPage()
  expect(await phone.evaluate(() => matchMedia('(pointer: coarse)').matches), 'the 390 px page is a touch screen').toBe(true)

  page.setDefaultTimeout(10_000)
  phone.setDefaultTimeout(10_000)
  for (const s of screens(role)) {
    const where = `${role} ${s.name}`
    await checked(`${where} at 1280`, async () => {
      await show(page, s)
      await axe(page, `${where} at 1280`)
      await tabWalk(page, `${where} at 1280`)
      if (s.dialogs) await s.dialogs(page, `${where} at 1280`, false)
    })
    await checked(`${where} at 390`, async () => {
      await show(phone, s)
      await axe(phone, `${where} at 390`)
      await targets(phone, `${where} at 390`)
      if (s.dialogs) await s.dialogs(phone, `${where} at 390`, true)
    })
  }
  report()
})
