import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { howItWorks } from '../../content/steps'
import { HowItWorks } from './HowItWorks'

afterEach(cleanup)

describe('HowItWorks', () => {
  it('renders the steps from content as a numbered list', () => {
    render(<HowItWorks />)

    expect(screen.getByRole('heading', { level: 2, name: howItWorks.heading })).toBeTruthy()
    const steps = within(screen.getByRole('list')).getAllByRole('listitem')
    expect(steps).toHaveLength(howItWorks.steps.length)
    howItWorks.steps.forEach((step, i) => {
      expect(within(steps[i]).getByText(`0${i + 1}`)).toBeTruthy()
      expect(within(steps[i]).getByRole('heading', { level: 3, name: step.title })).toBeTruthy()
      expect(within(steps[i]).getByText(step.body)).toBeTruthy()
    })
  })
})
