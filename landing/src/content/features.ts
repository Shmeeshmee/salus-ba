import { ids } from './sections'

export type FeatureIcon = 'webApp' | 'mobileApp' | 'video'

export type Feature = {
  icon: FeatureIcon
  title: string
  body: string
  dark: boolean
}

export const product = {
  id: ids.product,
  eyebrow: 'Product',
  heading: 'One platform. Two apps. No waiting room.',
  features: [
    {
      icon: 'webApp',
      title: 'For doctors and admins',
      body: 'A web app for schedules, the clinic calendar and a questionnaire builder. Only what you need on screen — the rest one click away.',
      dark: false,
    },
    {
      icon: 'mobileApp',
      title: 'For patients',
      body: 'A mobile app to browse clinics, book an appointment and fill in the pre-visit questionnaire before the doctor joins.',
      dark: false,
    },
    {
      icon: 'video',
      title: 'Consults that write themselves',
      body: 'HD video calls with live transcription, and AI that drafts the visit note for the doctor to review and sign.',
      dark: true,
    },
  ] satisfies Feature[],
}
