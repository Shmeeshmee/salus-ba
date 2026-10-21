import type { ReactNode } from 'react'
import type { FeatureIcon as IconName } from '../../content/features'

type FeatureIconProps = {
  name: IconName
  className: string
}

const PATHS: Record<IconName, ReactNode> = {
  webApp: (
    <>
      <rect x="3" y="5" width="22" height="15" rx="2.5" />
      <path d="M10 24h8M14 20v4M7 10h6M7 14h10" />
    </>
  ),
  mobileApp: (
    <>
      <rect x="8" y="2.5" width="12" height="23" rx="3" />
      <path d="M12.5 21.5h3M11.5 9l2 2 3.5-3.5" />
    </>
  ),
  video: (
    <>
      <rect x="2.5" y="7" width="16" height="14" rx="2.5" />
      <path d="M18.5 12l7-4v12l-7-4z" />
    </>
  ),
}

export function FeatureIcon({ name, className }: FeatureIconProps) {
  return (
    <svg
      className={className}
      width="28"
      height="28"
      viewBox="0 0 28 28"
      fill="none"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {PATHS[name]}
    </svg>
  )
}
