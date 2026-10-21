import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { pricing } from '../../content/pricing'
import { Pricing } from './Pricing'

afterEach(cleanup)

describe('Pricing', () => {
  it('renders the heading and every plan from content', () => {
    render(<Pricing />)

    expect(screen.getByRole('heading', { level: 2, name: pricing.heading })).toBeTruthy()
    for (const plan of pricing.plans) {
      expect(screen.getByText(plan.label)).toBeTruthy()
      expect(screen.getByText(plan.amount, { exact: false })).toBeTruthy()
      expect(screen.getByText(plan.body)).toBeTruthy()
    }
  })
})
