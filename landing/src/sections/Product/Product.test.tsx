import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { product } from '../../content/features'
import { Product } from './Product'

afterEach(cleanup)

describe('Product', () => {
  it('renders the heading and every feature from content', () => {
    render(<Product />)

    expect(screen.getByRole('heading', { level: 2, name: product.heading })).toBeTruthy()
    expect(screen.getAllByRole('article')).toHaveLength(product.features.length)
    for (const feature of product.features) {
      expect(screen.getByRole('heading', { level: 3, name: feature.title })).toBeTruthy()
      expect(screen.getByText(feature.body)).toBeTruthy()
    }
  })
})
