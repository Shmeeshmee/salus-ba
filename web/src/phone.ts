// groupPhone shows a stored phone number (E.164) as people in Bosnia and Herzegovina write it (wave 4 walk, finding 6):
// +387, the two-digit network or area code, then the subscriber number in threes, its last group taking a fourth
// digit when there is one (+387 61 000 399, +387 61 000 3991). Any other number is shown as stored: its grouping
// depends on a country this app does not know. Saving strips the spaces again.
export function groupPhone(phone: string): string {
  const m = /^\+387(\d{2})(\d*)$/.exec(phone)
  if (!m) return phone
  const [, code, rest] = m
  const groups = rest.length > 6 ? [rest.slice(0, 3), rest.slice(3)] : (rest.match(/\d{1,3}/g) ?? [])
  return ['+387', code, ...groups].join(' ')
}
