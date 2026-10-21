export type Palette = {
  bg: string
  ink: string
  accent: string
  tint: string
}

export type Specialty = {
  name: string
  palette: Palette
  doctor: string
  initials: string
  quote: string
  blurb: string
}

// The hero cycles through these; each one recolours the hero with its palette. Doctors and quotes are
// synthetic.
export const specialties: readonly Specialty[] = [
  {
    name: 'Cardiology',
    palette: { bg: '#F6E2DF', ink: '#2A100D', accent: '#B83A32', tint: '#F9D6D0' },
    doctor: 'Dr. Amra H.',
    initials: 'AH',
    quote: '“…tightness in the chest, mostly in the mornings.”',
    blurb: 'Follow-ups and test reviews without a trip to the clinic.',
  },
  {
    name: 'Dermatology',
    palette: { bg: '#F4E6D4', ink: '#2A1806', accent: '#A9561A', tint: '#F5DABD' },
    doctor: 'Dr. Lejla K.',
    initials: 'LK',
    quote: '“The rash spread after I changed creams.”',
    blurb: 'Patients share photos before the call, so the doctor starts with context.',
  },
  {
    name: 'Pediatrics',
    palette: { bg: '#DCE9F5', ink: '#0C1D31', accent: '#2560A0', tint: '#C9DDF1' },
    doctor: 'Dr. Emir S.',
    initials: 'ES',
    quote: '“The fever finally went down overnight.”',
    blurb: 'Parents get answers at 9 pm, without a waiting room full of coughs.',
  },
  {
    name: 'General practice',
    palette: { bg: '#DDF1E7', ink: '#0B271C', accent: '#0F7350', tint: '#C6E9D7' },
    doctor: 'Dr. Tarik M.',
    initials: 'TM',
    quote: '“Can I renew the same prescription?”',
    blurb: 'Renewals, results and quick questions in minutes, not days.',
  },
]
