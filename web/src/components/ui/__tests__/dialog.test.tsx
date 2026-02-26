/**
 * RAD Gateway Admin UI - Dialog Component Tests
 * Tests for Radix UI Dialog wrapper component
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from '../dialog';

// Mock @radix-ui/react-dialog's portal to render inline for testing
jest.mock('@radix-ui/react-dialog', () => {
  const actual = jest.requireActual('@radix-ui/react-dialog');
  return {
    ...actual,
    Portal: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  };
});

describe('Dialog Component', () => {
  it('should not render dialog content when closed', () => {
    render(
      <Dialog open={false}>
        <DialogContent>
          <DialogTitle>Hidden Title</DialogTitle>
          <div data-testid="dialog-content">Dialog Content</div>
        </DialogContent>
      </Dialog>
    );

    expect(screen.queryByTestId('dialog-content')).not.toBeInTheDocument();
  });

  it('should render dialog content when open', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
          <div data-testid="dialog-content">Dialog Content</div>
        </DialogContent>
      </Dialog>
    );

    expect(screen.getByTestId('dialog-content')).toBeInTheDocument();
  });

  it('should open dialog when DialogTrigger is clicked', async () => {
    const user = userEvent.setup();

    render(
      <Dialog>
        <DialogTrigger data-testid="dialog-trigger">Open Dialog</DialogTrigger>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
          <div data-testid="dialog-content">Dialog Content</div>
        </DialogContent>
      </Dialog>
    );

    // Initially closed
    expect(screen.queryByTestId('dialog-content')).not.toBeInTheDocument();

    // Click trigger to open
    await user.click(screen.getByTestId('dialog-trigger'));

    await waitFor(() => {
      expect(screen.getByTestId('dialog-content')).toBeInTheDocument();
    });
  });

  it('should close dialog when DialogClose is clicked', async () => {
    const user = userEvent.setup();

    render(
      <Dialog defaultOpen>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
          <div data-testid="dialog-content">Dialog Content</div>
          <DialogClose data-testid="dialog-close">Close</DialogClose>
        </DialogContent>
      </Dialog>
    );

    // Initially open
    expect(screen.getByTestId('dialog-content')).toBeInTheDocument();

    // Click close button
    await user.click(screen.getByTestId('dialog-close'));

    await waitFor(() => {
      expect(screen.queryByTestId('dialog-content')).not.toBeInTheDocument();
    });
  });

  it('should render DialogHeader correctly', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogHeader data-testid="dialog-header">
            <DialogTitle>Test Title</DialogTitle>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    );

    const header = screen.getByTestId('dialog-header');
    expect(header).toBeInTheDocument();
    expect(header.tagName.toLowerCase()).toBe('div');
  });

  it('should render DialogTitle correctly with proper attributes', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle data-testid="dialog-title">Test Title</DialogTitle>
        </DialogContent>
      </Dialog>
    );

    const title = screen.getByTestId('dialog-title');
    expect(title).toBeInTheDocument();
    expect(title).toHaveTextContent('Test Title');
  });

  it('should render DialogDescription correctly', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
          <DialogDescription data-testid="dialog-description">
            Test Description
          </DialogDescription>
        </DialogContent>
      </Dialog>
    );

    const description = screen.getByTestId('dialog-description');
    expect(description).toBeInTheDocument();
    expect(description).toHaveTextContent('Test Description');
  });

  it('should render DialogFooter correctly', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
          <DialogFooter data-testid="dialog-footer">
            <button>Action</button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );

    const footer = screen.getByTestId('dialog-footer');
    expect(footer).toBeInTheDocument();
    expect(footer.tagName.toLowerCase()).toBe('div');
  });

  it('should call onOpenChange callback when dialog opens', async () => {
    const user = userEvent.setup();
    const onOpenChange = jest.fn();

    render(
      <Dialog onOpenChange={onOpenChange}>
        <DialogTrigger data-testid="dialog-trigger">Open Dialog</DialogTrigger>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
          <div>Dialog Content</div>
        </DialogContent>
      </Dialog>
    );

    await user.click(screen.getByTestId('dialog-trigger'));

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(true);
    });
  });

  it('should call onOpenChange callback when dialog closes', async () => {
    const user = userEvent.setup();
    const onOpenChange = jest.fn();

    render(
      <Dialog open={true} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
          <div>Dialog Content</div>
          <DialogClose data-testid="dialog-close">Close</DialogClose>
        </DialogContent>
      </Dialog>
    );

    await user.click(screen.getByTestId('dialog-close'));

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it('should render complete dialog with all sub-components', () => {
    render(
      <Dialog open={true}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Dialog Title</DialogTitle>
            <DialogDescription>Dialog Description</DialogDescription>
          </DialogHeader>
          <div>Main content</div>
          <DialogFooter>
            <button>Cancel</button>
            <button>Confirm</button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );

    expect(screen.getByText('Dialog Title')).toBeInTheDocument();
    expect(screen.getByText('Dialog Description')).toBeInTheDocument();
    expect(screen.getByText('Main content')).toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
    expect(screen.getByText('Confirm')).toBeInTheDocument();
  });

  it('should handle controlled state correctly', async () => {
    const user = userEvent.setup();
    const onOpenChange = jest.fn();

    function ControlledDialog() {
      const [open, setOpen] = React.useState(false);

      return (
        <Dialog open={open} onOpenChange={(isOpen) => {
          setOpen(isOpen);
          onOpenChange(isOpen);
        }}>
          <DialogTrigger data-testid="dialog-trigger">Open</DialogTrigger>
          <DialogContent>
            <DialogTitle>Test Title</DialogTitle>
            <div data-testid="dialog-content">Content</div>
            <DialogClose data-testid="dialog-close">Close</DialogClose>
          </DialogContent>
        </Dialog>
      );
    }

    render(<ControlledDialog />);

    // Initially closed
    expect(screen.queryByTestId('dialog-content')).not.toBeInTheDocument();

    // Open dialog
    await user.click(screen.getByTestId('dialog-trigger'));
    await waitFor(() => {
      expect(screen.getByTestId('dialog-content')).toBeInTheDocument();
    });
    expect(onOpenChange).toHaveBeenCalledWith(true);

    // Close dialog
    await user.click(screen.getByTestId('dialog-close'));
    await waitFor(() => {
      expect(screen.queryByTestId('dialog-content')).not.toBeInTheDocument();
    });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('should have close button in DialogContent', () => {
    render(
      <Dialog open={true}>
        <DialogContent data-testid="dialog-content-wrapper">
          <DialogTitle>Test Title</DialogTitle>
          <div>Content</div>
        </DialogContent>
      </Dialog>
    );

    // The close button is rendered by DialogContent with sr-only text
    expect(screen.getByText('Close')).toBeInTheDocument();
  });

  it('should render with uncontrolled defaultOpen state', () => {
    render(
      <Dialog defaultOpen>
        <DialogContent>
          <DialogTitle>Test Title</DialogTitle>
          <div data-testid="dialog-content">Content</div>
        </DialogContent>
      </Dialog>
    );

    expect(screen.getByTestId('dialog-content')).toBeInTheDocument();
  });
});
