// A few hand-drawn icons, 20 px on a 20 px grid, drawn with the current colour.
const stroke = {
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.6,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
} as const

export function Logo() {
  return (
    <span className="logo">
      <svg width="24" height="24" viewBox="0 0 24 24" aria-hidden="true">
        <rect width="24" height="24" rx="7" fill="var(--color-brand)" />
        <path d="M12 6.5v11M6.5 12h11" stroke="var(--color-mint)" strokeWidth="2.6" strokeLinecap="round" />
      </svg>
      Salus
    </span>
  )
}

export function PeopleIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <circle cx="7.5" cy="7" r="2.8" />
      <path d="M2.5 16c.6-2.7 2.6-4.2 5-4.2s4.4 1.5 5 4.2" />
      <path d="M13 4.6a2.6 2.6 0 0 1 0 4.9M14.6 11.9c1.5.5 2.5 1.8 2.9 4.1" />
    </svg>
  )
}

export function ClinicIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d="M3.5 17V7.5L10 3l6.5 4.5V17z" />
      <path d="M10 8.5v5M7.5 11h5" />
    </svg>
  )
}

export function SignOutIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d="M8 3.5H4.5v13H8M12.5 6.5 16 10l-3.5 3.5M16 10H8" />
    </svg>
  )
}
