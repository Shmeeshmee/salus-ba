// Fixed messages shared by the screens. API error codes are mapped to plain sentences; nothing from a
// response body is shown.
export const somethingWrong = 'Something went wrong. Try again.'
export const wrongCode = 'The code is wrong. Check the time on your device and try again.'

export function tooManyAttempts(retryAfter: string | null): string {
  const seconds = Number(retryAfter)
  if (!Number.isFinite(seconds) || seconds <= 0) return 'Too many attempts. Try again later.'
  const minutes = Math.ceil(seconds / 60)
  return `Too many attempts. Try again in ${minutes} ${minutes === 1 ? 'minute' : 'minutes'}.`
}

export const statusLabels = { booked: 'Booked', cancelled: 'Cancelled', fulfilled: 'Fulfilled', noshow: 'No-show' } as const
