import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import { Button } from '../button'

describe('Button Component', () => {
  describe('Rendering', () => {
    it('renders with default props', () => {
      render(<Button>Click me</Button>)
      const button = screen.getByRole('button', { name: /click me/i })
      expect(button).toBeInTheDocument()
      expect(button).toHaveClass('inline-flex')
      expect(button).toHaveClass('items-center')
      expect(button).toHaveClass('justify-center')
    })

    it('renders with custom text', () => {
      render(<Button>Custom Button</Button>)
      expect(screen.getByRole('button', { name: /custom button/i })).toBeInTheDocument()
    })

    it('renders with custom className', () => {
      render(<Button className="custom-class">Click me</Button>)
      const button = screen.getByRole('button', { name: /click me/i })
      expect(button).toHaveClass('custom-class')
    })
  })

  describe('Variant Classes', () => {
    it('renders with default variant', () => {
      render(<Button variant="default">Default</Button>)
      const button = screen.getByRole('button', { name: /default/i })
      expect(button).toHaveClass('bg-primary')
      expect(button).toHaveClass('text-primary-foreground')
    })

    it('renders with destructive variant', () => {
      render(<Button variant="destructive">Destructive</Button>)
      const button = screen.getByRole('button', { name: /destructive/i })
      expect(button).toHaveClass('bg-destructive')
      expect(button).toHaveClass('text-destructive-foreground')
    })

    it('renders with outline variant', () => {
      render(<Button variant="outline">Outline</Button>)
      const button = screen.getByRole('button', { name: /outline/i })
      expect(button).toHaveClass('border')
      expect(button).toHaveClass('border-input')
      expect(button).toHaveClass('bg-background')
    })

    it('renders with secondary variant', () => {
      render(<Button variant="secondary">Secondary</Button>)
      const button = screen.getByRole('button', { name: /secondary/i })
      expect(button).toHaveClass('bg-secondary')
      expect(button).toHaveClass('text-secondary-foreground')
    })

    it('renders with ghost variant', () => {
      render(<Button variant="ghost">Ghost</Button>)
      const button = screen.getByRole('button', { name: /ghost/i })
      expect(button).toHaveClass('hover:bg-accent')
    })

    it('renders with link variant', () => {
      render(<Button variant="link">Link</Button>)
      const button = screen.getByRole('button', { name: /link/i })
      expect(button).toHaveClass('text-primary')
      expect(button).toHaveClass('underline-offset-4')
    })
  })

  describe('Size Classes', () => {
    it('renders with default size', () => {
      render(<Button size="default">Default Size</Button>)
      const button = screen.getByRole('button', { name: /default size/i })
      expect(button).toHaveClass('h-9')
      expect(button).toHaveClass('px-4')
      expect(button).toHaveClass('py-2')
    })

    it('renders with sm size', () => {
      render(<Button size="sm">Small</Button>)
      const button = screen.getByRole('button', { name: /small/i })
      expect(button).toHaveClass('h-8')
      expect(button).toHaveClass('px-3')
      expect(button).toHaveClass('text-xs')
    })

    it('renders with lg size', () => {
      render(<Button size="lg">Large</Button>)
      const button = screen.getByRole('button', { name: /large/i })
      expect(button).toHaveClass('h-10')
      expect(button).toHaveClass('px-8')
    })

    it('renders with icon size', () => {
      render(<Button size="icon">Icon</Button>)
      const button = screen.getByRole('button', { name: /icon/i })
      expect(button).toHaveClass('h-9')
      expect(button).toHaveClass('w-9')
    })
  })

  describe('Click Handlers', () => {
    it('calls onClick handler when clicked', () => {
      const handleClick = jest.fn()
      render(<Button onClick={handleClick}>Click me</Button>)
      const button = screen.getByRole('button', { name: /click me/i })
      fireEvent.click(button)
      expect(handleClick).toHaveBeenCalledTimes(1)
    })

    it('calls onClick handler multiple times', () => {
      const handleClick = jest.fn()
      render(<Button onClick={handleClick}>Click me</Button>)
      const button = screen.getByRole('button', { name: /click me/i })
      fireEvent.click(button)
      fireEvent.click(button)
      fireEvent.click(button)
      expect(handleClick).toHaveBeenCalledTimes(3)
    })

    it('receives click event in handler', () => {
      const handleClick = jest.fn()
      render(<Button onClick={handleClick}>Click me</Button>)
      const button = screen.getByRole('button', { name: /click me/i })
      fireEvent.click(button)
      expect(handleClick).toHaveBeenCalledWith(expect.any(Object))
    })
  })

  describe('Disabled State', () => {
    it('renders as disabled when disabled prop is true', () => {
      render(<Button disabled>Disabled</Button>)
      const button = screen.getByRole('button', { name: /disabled/i })
      expect(button).toBeDisabled()
      expect(button).toHaveAttribute('disabled')
    })

    it('has disabled styling classes', () => {
      render(<Button disabled>Disabled</Button>)
      const button = screen.getByRole('button', { name: /disabled/i })
      expect(button).toHaveClass('disabled:pointer-events-none')
      expect(button).toHaveClass('disabled:opacity-50')
    })

    it('does not call onClick when disabled', () => {
      const handleClick = jest.fn()
      render(<Button disabled onClick={handleClick}>Disabled</Button>)
      const button = screen.getByRole('button', { name: /disabled/i })
      fireEvent.click(button)
      expect(handleClick).not.toHaveBeenCalled()
    })
  })

  describe('asChild Prop', () => {
    it('renders as button by default', () => {
      render(<Button>Default Button</Button>)
      const button = screen.getByRole('button', { name: /default button/i })
      expect(button.tagName).toBe('BUTTON')
    })

    it('renders children when asChild is true', () => {
      render(
        <Button asChild>
          <a href="/test">Link as Button</a>
        </Button>
      )
      const link = screen.getByRole('link', { name: /link as button/i })
      expect(link).toBeInTheDocument()
      expect(link.tagName).toBe('A')
      expect(link).toHaveAttribute('href', '/test')
    })

    it('applies button classes to child element when asChild is true', () => {
      render(
        <Button asChild variant="destructive">
          <span>Span as Button</span>
        </Button>
      )
      const span = screen.getByText(/span as button/i)
      expect(span).toHaveClass('bg-destructive')
      expect(span).toHaveClass('text-destructive-foreground')
    })

    it('renders complex child with asChild', () => {
      render(
        <Button asChild>
          <div data-testid="custom-div">
            <span>Nested Content</span>
          </div>
        </Button>
      )
      const div = screen.getByTestId('custom-div')
      expect(div).toBeInTheDocument()
      expect(div).toHaveClass('inline-flex')
    })
  })

  describe('Type Attribute', () => {
    it('renders button element by default', () => {
      render(<Button>Type Test</Button>)
      const button = screen.getByRole('button', { name: /type test/i })
      expect(button.tagName).toBe('BUTTON')
    })

    it('renders with type="submit" when specified', () => {
      render(<Button type="submit">Submit</Button>)
      const button = screen.getByRole('button', { name: /submit/i })
      expect(button).toHaveAttribute('type', 'submit')
    })

    it('renders with type="reset" when specified', () => {
      render(<Button type="reset">Reset</Button>)
      const button = screen.getByRole('button', { name: /reset/i })
      expect(button).toHaveAttribute('type', 'reset')
    })
  })

  describe('Ref Forwarding', () => {
    it('forwards ref to button element', () => {
      const ref = React.createRef<HTMLButtonElement>()
      render(<Button ref={ref}>Ref Test</Button>)
      expect(ref.current).toBeInstanceOf(HTMLButtonElement)
      expect(ref.current?.tagName).toBe('BUTTON')
    })

    it('ref has focus method', () => {
      const ref = React.createRef<HTMLButtonElement>()
      render(<Button ref={ref}>Ref Test</Button>)
      expect(typeof ref.current?.focus).toBe('function')
    })
  })

  describe('Additional Props', () => {
    it('passes through data attributes', () => {
      render(<Button data-testid="custom-button">Custom Data</Button>)
      expect(screen.getByTestId('custom-button')).toBeInTheDocument()
    })

    it('passes through aria attributes', () => {
      render(<Button aria-label="Custom Label">Aria</Button>)
      expect(screen.getByLabelText(/custom label/i)).toBeInTheDocument()
    })

    it('passes through id attribute', () => {
      render(<Button id="my-button">ID Test</Button>)
      expect(screen.getByRole('button', { name: /id test/i })).toHaveAttribute('id', 'my-button')
    })

    it('passes through name attribute', () => {
      render(<Button name="button-name">Name Test</Button>)
      expect(screen.getByRole('button', { name: /name test/i })).toHaveAttribute('name', 'button-name')
    })
  })

  describe('Combined Props', () => {
    it('renders with variant, size, and custom class', () => {
      render(
        <Button variant="outline" size="lg" className="extra-class">
          Combined Props
        </Button>
      )
      const button = screen.getByRole('button', { name: /combined props/i })
      expect(button).toHaveClass('border')
      expect(button).toHaveClass('h-10')
      expect(button).toHaveClass('px-8')
      expect(button).toHaveClass('extra-class')
    })

    it('renders disabled with variant', () => {
      render(
        <Button variant="destructive" disabled>
          Disabled Destructive
        </Button>
      )
      const button = screen.getByRole('button', { name: /disabled destructive/i })
      expect(button).toBeDisabled()
      expect(button).toHaveClass('bg-destructive')
    })

    it('renders asChild with size variant', () => {
      render(
        <Button asChild size="sm">
          <a href="/">Small Link</a>
        </Button>
      )
      const link = screen.getByRole('link', { name: /small link/i })
      expect(link).toHaveClass('h-8')
      expect(link).toHaveClass('px-3')
      expect(link).toHaveClass('text-xs')
    })
  })
})
