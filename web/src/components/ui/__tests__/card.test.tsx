/**
 * Card Component Unit Tests
 * RAD Gateway Admin UI - UI Modernization Sprint
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from '../card';

describe('Card Component', () => {
  describe('Card', () => {
    it('renders with correct structure', () => {
      render(<Card data-testid="card">Card Content</Card>);
      const card = screen.getByTestId('card');
      expect(card).toBeInTheDocument();
      expect(card.tagName).toBe('DIV');
    });

    it('renders children correctly', () => {
      render(
        <Card data-testid="card">
          <span data-testid="child">Child Content</span>
        </Card>
      );
      expect(screen.getByTestId('child')).toBeInTheDocument();
      expect(screen.getByText('Child Content')).toBeInTheDocument();
    });

    it('applies default classes', () => {
      render(<Card data-testid="card">Content</Card>);
      const card = screen.getByTestId('card');
      expect(card).toHaveClass('rounded-xl');
      expect(card).toHaveClass('border');
      expect(card).toHaveClass('bg-card');
      expect(card).toHaveClass('text-card-foreground');
      expect(card).toHaveClass('shadow');
    });

    it('merges custom className with default classes', () => {
      render(<Card data-testid="card" className="custom-class">Content</Card>);
      const card = screen.getByTestId('card');
      expect(card).toHaveClass('custom-class');
      expect(card).toHaveClass('rounded-xl');
    });

    it('forwards ref correctly', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<Card ref={ref}>Content</Card>);
      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });

    it('has correct displayName', () => {
      expect(Card.displayName).toBe('Card');
    });
  });

  describe('CardHeader', () => {
    it('renders correctly', () => {
      render(<CardHeader data-testid="header">Header Content</CardHeader>);
      const header = screen.getByTestId('header');
      expect(header).toBeInTheDocument();
      expect(header.tagName).toBe('DIV');
    });

    it('applies default classes', () => {
      render(<CardHeader data-testid="header">Content</CardHeader>);
      const header = screen.getByTestId('header');
      expect(header).toHaveClass('flex');
      expect(header).toHaveClass('flex-col');
      expect(header).toHaveClass('space-y-1.5');
      expect(header).toHaveClass('p-6');
    });

    it('merges custom className', () => {
      render(<CardHeader data-testid="header" className="custom-header">Content</CardHeader>);
      const header = screen.getByTestId('header');
      expect(header).toHaveClass('custom-header');
      expect(header).toHaveClass('flex');
    });

    it('forwards ref correctly', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<CardHeader ref={ref}>Content</CardHeader>);
      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });

    it('has correct displayName', () => {
      expect(CardHeader.displayName).toBe('CardHeader');
    });
  });

  describe('CardTitle', () => {
    it('renders correctly', () => {
      render(<CardTitle data-testid="title">Card Title</CardTitle>);
      const title = screen.getByTestId('title');
      expect(title).toBeInTheDocument();
      expect(title.tagName).toBe('DIV');
    });

    it('applies default classes', () => {
      render(<CardTitle data-testid="title">Title</CardTitle>);
      const title = screen.getByTestId('title');
      expect(title).toHaveClass('font-semibold');
      expect(title).toHaveClass('leading-none');
      expect(title).toHaveClass('tracking-tight');
    });

    it('merges custom className', () => {
      render(<CardTitle data-testid="title" className="custom-title">Title</CardTitle>);
      const title = screen.getByTestId('title');
      expect(title).toHaveClass('custom-title');
      expect(title).toHaveClass('font-semibold');
    });

    it('forwards ref correctly', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<CardTitle ref={ref}>Title</CardTitle>);
      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });

    it('has correct displayName', () => {
      expect(CardTitle.displayName).toBe('CardTitle');
    });
  });

  describe('CardDescription', () => {
    it('renders correctly', () => {
      render(<CardDescription data-testid="description">Description text</CardDescription>);
      const description = screen.getByTestId('description');
      expect(description).toBeInTheDocument();
      expect(description.tagName).toBe('DIV');
    });

    it('applies default classes', () => {
      render(<CardDescription data-testid="description">Description</CardDescription>);
      const description = screen.getByTestId('description');
      expect(description).toHaveClass('text-sm');
      expect(description).toHaveClass('text-muted-foreground');
    });

    it('merges custom className', () => {
      render(<CardDescription data-testid="description" className="custom-desc">Description</CardDescription>);
      const description = screen.getByTestId('description');
      expect(description).toHaveClass('custom-desc');
      expect(description).toHaveClass('text-sm');
    });

    it('forwards ref correctly', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<CardDescription ref={ref}>Description</CardDescription>);
      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });

    it('has correct displayName', () => {
      expect(CardDescription.displayName).toBe('CardDescription');
    });
  });

  describe('CardContent', () => {
    it('renders correctly', () => {
      render(<CardContent data-testid="content">Content text</CardContent>);
      const content = screen.getByTestId('content');
      expect(content).toBeInTheDocument();
      expect(content.tagName).toBe('DIV');
    });

    it('applies default classes', () => {
      render(<CardContent data-testid="content">Content</CardContent>);
      const content = screen.getByTestId('content');
      expect(content).toHaveClass('p-6');
      expect(content).toHaveClass('pt-0');
    });

    it('merges custom className', () => {
      render(<CardContent data-testid="content" className="custom-content">Content</CardContent>);
      const content = screen.getByTestId('content');
      expect(content).toHaveClass('custom-content');
      expect(content).toHaveClass('p-6');
    });

    it('forwards ref correctly', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<CardContent ref={ref}>Content</CardContent>);
      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });

    it('has correct displayName', () => {
      expect(CardContent.displayName).toBe('CardContent');
    });
  });

  describe('CardFooter', () => {
    it('renders correctly', () => {
      render(<CardFooter data-testid="footer">Footer Content</CardFooter>);
      const footer = screen.getByTestId('footer');
      expect(footer).toBeInTheDocument();
      expect(footer.tagName).toBe('DIV');
    });

    it('applies default classes', () => {
      render(<CardFooter data-testid="footer">Footer</CardFooter>);
      const footer = screen.getByTestId('footer');
      expect(footer).toHaveClass('flex');
      expect(footer).toHaveClass('items-center');
      expect(footer).toHaveClass('p-6');
      expect(footer).toHaveClass('pt-0');
    });

    it('merges custom className', () => {
      render(<CardFooter data-testid="footer" className="custom-footer">Footer</CardFooter>);
      const footer = screen.getByTestId('footer');
      expect(footer).toHaveClass('custom-footer');
      expect(footer).toHaveClass('flex');
    });

    it('forwards ref correctly', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<CardFooter ref={ref}>Footer</CardFooter>);
      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });

    it('has correct displayName', () => {
      expect(CardFooter.displayName).toBe('CardFooter');
    });
  });

  describe('Nested Components', () => {
    it('works together as a complete card structure', () => {
      render(
        <Card data-testid="card">
          <CardHeader data-testid="header">
            <CardTitle data-testid="title">Card Title</CardTitle>
            <CardDescription data-testid="description">Card Description</CardDescription>
          </CardHeader>
          <CardContent data-testid="content">Card Content</CardContent>
          <CardFooter data-testid="footer">Card Footer</CardFooter>
        </Card>
      );

      expect(screen.getByTestId('card')).toBeInTheDocument();
      expect(screen.getByTestId('header')).toBeInTheDocument();
      expect(screen.getByTestId('title')).toBeInTheDocument();
      expect(screen.getByTestId('description')).toBeInTheDocument();
      expect(screen.getByTestId('content')).toBeInTheDocument();
      expect(screen.getByTestId('footer')).toBeInTheDocument();
    });

    it('renders complete card with all content visible', () => {
      render(
        <Card>
          <CardHeader>
            <CardTitle>Test Title</CardTitle>
            <CardDescription>Test Description</CardDescription>
          </CardHeader>
          <CardContent>Test Content</CardContent>
          <CardFooter>Test Footer</CardFooter>
        </Card>
      );

      expect(screen.getByText('Test Title')).toBeInTheDocument();
      expect(screen.getByText('Test Description')).toBeInTheDocument();
      expect(screen.getByText('Test Content')).toBeInTheDocument();
      expect(screen.getByText('Test Footer')).toBeInTheDocument();
    });

    it('maintains proper nesting structure', () => {
      const { container } = render(
        <Card>
          <CardHeader>
            <CardTitle>Title</CardTitle>
          </CardHeader>
          <CardContent>Content</CardContent>
          <CardFooter>Footer</CardFooter>
        </Card>
      );

      const card = container.firstChild as HTMLElement;
      expect(card.children.length).toBe(3);
      expect(card.children[0].tagName).toBe('DIV'); // CardHeader
      expect(card.children[1].tagName).toBe('DIV'); // CardContent
      expect(card.children[2].tagName).toBe('DIV'); // CardFooter
    });

    it('allows custom classes at each level', () => {
      render(
        <Card className="custom-card">
          <CardHeader className="custom-header">
            <CardTitle className="custom-title">Title</CardTitle>
            <CardDescription className="custom-desc">Description</CardDescription>
          </CardHeader>
          <CardContent className="custom-content">Content</CardContent>
          <CardFooter className="custom-footer">Footer</CardFooter>
        </Card>
      );

      expect(screen.getByText('Title').parentElement).toHaveClass('custom-header');
      expect(screen.getByText('Title')).toHaveClass('custom-title');
      expect(screen.getByText('Description')).toHaveClass('custom-desc');
      expect(screen.getByText('Content')).toHaveClass('custom-content');
      expect(screen.getByText('Footer')).toHaveClass('custom-footer');
    });
  });
});
