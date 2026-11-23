import { ids } from './sections'

export type FeatureIcon = 'receipt' | 'webApp' | 'mobileApp' | 'video'

export type Feature = {
  icon: FeatureIcon
  title: string
  body: string
  dark: boolean
}

export const product = {
  id: ids.product,
  eyebrow: 'Product',
  heading: 'Everything the clinic runs on, in one system.',
  features: [
    {
      icon: 'receipt',
      title: 'At the front desk',
      body: 'In-person and video visits in one calendar, your price list, and receipts issued from it. Built for the new FBiH fiscalization law; reporting to the Tax Administration through ESET is coming.',
      dark: false,
    },
    {
      icon: 'webApp',
      title: 'For doctors and admins',
      body: 'A web app for schedules, the clinic calendar, the questionnaire builder, patient documents, chat with patients and analytics. Only what you need on screen, the rest one click away.',
      dark: false,
    },
    {
      icon: 'mobileApp',
      title: 'For patients',
      body: 'A mobile app to find the clinic, book a visit, get reminders, join the waitlist for an earlier time and fill in the questionnaire before the visit.',
      dark: false,
    },
    {
      icon: 'video',
      title: 'Online hours, from home',
      body: 'Your doctors can also see patients by video after hours, with live transcription and AI that drafts the visit note for the doctor to review and sign. More hours for the clinic, without more rooms or staff.',
      dark: true,
    },
  ] satisfies Feature[],
}
