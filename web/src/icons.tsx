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

export function CalendarIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <rect x="3" y="4.5" width="14" height="12.5" rx="2.5" />
      <path d="M3 8.5h14M7 3v3M13 3v3" />
    </svg>
  )
}

export function ClockIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <circle cx="10" cy="10" r="7" />
      <path d="M10 6v4l2.5 2" />
    </svg>
  )
}

export function ChevronIcon({ direction }: { direction: 'left' | 'right' }) {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d={direction === 'left' ? 'M12 5l-5 5 5 5' : 'M8 5l5 5-5 5'} />
    </svg>
  )
}

export function CrossIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d="M6 6l8 8M14 6l-8 8" />
    </svg>
  )
}

// Call controls: a slash across the icon when it is off.
export function MicrophoneIcon({ off = false }: { off?: boolean }) {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <rect x="7.5" y="2.5" width="5" height="9" rx="2.5" />
      <path d="M4.5 9.5a5.5 5.5 0 0 0 11 0M10 15v2.5" />
      {off && <path d="M3.5 3.5l13 13" />}
    </svg>
  )
}

export function CameraIcon({ off = false }: { off?: boolean }) {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <rect x="2.5" y="5.5" width="10.5" height="9" rx="2" />
      <path d="M13 9l4.5-2.5v7L13 11" />
      {off && <path d="M3.5 3.5l13 13" />}
    </svg>
  )
}

export function LeaveIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d="M11 3.5H5.5v13H11M14 7l3 3-3 3M17 10H9" />
    </svg>
  )
}
