import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ProgressBar } from './ProgressBar'

describe('ProgressBar', () => {
  it('renderiza barra com 0%', () => {
    render(<ProgressBar current={0} target={100} />)
    expect(screen.getByText('0%')).toBeInTheDocument()
  })

  it('renderiza barra com 50%', () => {
    render(<ProgressBar current={50} target={100} />)
    expect(screen.getByText('50%')).toBeInTheDocument()
  })

  it('renderiza barra com 100%', () => {
    render(<ProgressBar current={100} target={100} />)
    expect(screen.getByText('100%')).toBeInTheDocument()
  })

  it('renderiza barra com status sucesso (100%+)', () => {
    const { container } = render(<ProgressBar current={150} target={100} />)
    expect(container.querySelector('.progress-bar.success')).toBeInTheDocument()
  })

  it('renderiza barra com status aviso (70%-99%)', () => {
    const { container } = render(<ProgressBar current={80} target={100} />)
    expect(container.querySelector('.progress-bar.warning')).toBeInTheDocument()
  })

  it('renderiza barra com status perigo (<70%)', () => {
    const { container } = render(<ProgressBar current={50} target={100} />)
    expect(container.querySelector('.progress-bar.danger')).toBeInTheDocument()
  })

  it('mostra label quando fornecido', () => {
    render(<ProgressBar current={50} target={100} label="Aquisições" />)
    expect(screen.getByText('Aquisições')).toBeInTheDocument()
  })

  it('mostra estatísticas (current/target)', () => {
    render(<ProgressBar current={50} target={100} />)
    expect(screen.getByText('50')).toBeInTheDocument()
    expect(screen.getByText('100')).toBeInTheDocument()
  })

  it('não mostra percentagem quando showPercentage=false', () => {
    render(<ProgressBar current={50} target={100} showPercentage={false} />)
    // Ainda mostra as estatísticas, mas não a percentagem
    expect(screen.queryByText('50%')).not.toBeInTheDocument()
  })
})
