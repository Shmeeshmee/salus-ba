import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { pricing } from '../../content/pricing'
import { Pricing } from './Pricing'

afterEach(cleanup)

describe('Pricing', () => {
  it('renders the heading, every plan and the note from content', () => {
    render(<Pricing />)

    expect(screen.getByRole('heading', { level: 2, name: pricing.heading })).toBeTruthy()
    for (const plan of pricing.plans) {
      expect(screen.getByText(plan.label)).toBeTruthy()
      expect(screen.getByText(plan.body)).toBeTruthy()
    }
    expect(screen.getAllByText(pricing.plans[0].amount, { exact: false })).toHaveLength(pricing.plans.length)
    for (const line of pricing.notes) {
      expect(screen.getByText(line)).toBeTruthy()
    }
  })

  it('prices two plans per practitioner per month, with the price still a placeholder', () => {
    expect(pricing.plans).toHaveLength(2)
    for (const plan of pricing.plans) {
      expect(plan.amount).toBe('[PRICE]')
      expect(plan.unit).toBe(' / practitioner / month')
    }
    const copy = JSON.stringify(pricing).toLowerCase()
    expect(copy).not.toMatch(/year|%|share/)
    expect(copy).toContain('30-day trial')
  })
})
