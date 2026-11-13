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

// WaitlistIcon is a queue: three people's places, the first one ahead (SAL-35).
export function WaitlistIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d="M3.5 5.5h13M3.5 10h9M3.5 14.5h9" />
      <path d="M15.5 11.5 17 13l-1.5 1.5" />
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

export function ListIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <rect x="4" y="3" width="12" height="14.5" rx="2" />
      <path d="M7.5 7.5h5M7.5 10.5h5M7.5 13.5h3" />
    </svg>
  )
}

export function GripIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" fill="currentColor">
      {[6, 10, 14].flatMap((y) => [7.5, 12.5].map((x) => <circle key={`${x}-${y}`} cx={x} cy={y} r="1.3" />))}
    </svg>
  )
}

export function ArrowIcon({ direction }: { direction: 'up' | 'down' }) {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d={direction === 'up' ? 'M10 15.5v-11M5.5 9 10 4.5 14.5 9' : 'M10 4.5v11M5.5 11 10 15.5 14.5 11'} />
    </svg>
  )
}

// A sheet with a tick: the term review.
export function ReviewIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <rect x="4" y="3" width="12" height="14.5" rx="2" />
      <path d="M7.5 10.5l2 2 3.5-4" />
    </svg>
  )
}

// A speech bubble: the messages.
export function ChatIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d="M4.5 4h11a2 2 0 0 1 2 2v6.5a2 2 0 0 1-2 2H9l-3.5 2.5v-2.5h-1a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z" />
    </svg>
  )
}

// A paper clip: attach a file.
export function PaperclipIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" {...stroke}>
      <path d="M15.5 9.5l-5.8 5.8a3.5 3.5 0 0 1-5-5l6.3-6.3a2.3 2.3 0 0 1 3.3 3.3l-6.2 6.2a1.1 1.1 0 0 1-1.6-1.6l5.6-5.6" />
    </svg>
  )
}
