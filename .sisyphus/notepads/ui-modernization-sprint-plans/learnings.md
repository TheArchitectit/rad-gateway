## 2026-02-25 Task 1.1: Initialize shadcn/ui

### Learnings
- shadcn/ui CLI initializes cleanly with Next.js 14 + Tailwind 3.3
- Type errors common with Toaster components - explicit type casting required for theme prop
- Fix: `theme={theme as "light" | "dark" | "system"}` resolves TypeScript strict mode errors
- Build succeeds after fixing sonner.tsx type cast
- 15 components installed: button, card, dialog, input, label, select, checkbox, switch, table, tabs, tooltip, badge, skeleton, sonner, command
- Components created in web/src/components/ui/ directory
- tailwind.config.ts updated with shadcn configuration
- globals.css updated with shadcn CSS variables
- utils.ts created with cn() helper (tailwind-merge + clsx)

### Issues
- TypeScript strict mode caught theme type mismatch (undefined not assignable to "light" | "dark" | "system")
- Resolved by explicit type casting on line 13 of sonner.tsx

### Dependencies Installed
- tailwindcss-animate (required for animations)
- class-variance-authority (for component variants)
- clsx + tailwind-merge (for cn() utility)
- @radix-ui/react-* (installed automatically by shadcn CLI)

### Next Steps
- Task 1.2: Map Art Deco colors (brass/copper/steel) to shadcn theme
- Task 1.3: Dashboard redesign with KPI cards
- Task 1.4: Real-time metrics integration (uses existing SSE hooks)

## 2026-02-25 Task 1.6: Provider Health Dashboard

### Completed
- Created `web/src/components/providers/provider-health-card.tsx` with:
  - Art Deco styling with brass/copper accents
  - Status indicator with pulse animation
  - Provider icon/logo (OpenAI, Anthropic, Gemini, generic)
  - Success rate progress bar
  - Circuit breaker indicator
  - Latency metrics
  - 24h sparkline via ProviderStatusTimeline
  - Quick actions (retry, disable)

- Created `web/src/components/providers/provider-status-timeline.tsx` with:
  - SVG sparkline chart showing 24h uptime
  - Smooth cubic bezier curves
  - Color coding (green/yellow/red/gray)
  - Area gradient fill
  - Hover tooltip with exact value
  - Average reference line
  - Responsive design

- Refactored `web/src/app/providers/page.tsx` with:
  - Provider cards grid (1 col mobile, 2 col tablet, 3 col desktop)
  - Alerts section for failing providers
  - Filters (status, model type)
  - Search functionality
  - Refresh button
  - Stats cards (total, healthy, degraded, unhealthy)
  - View toggle (grid/list)
  - Empty state
  - Error state

### TypeScript Lessons
- Strict mode requires non-null assertions for array access: `points[0]!`
- Template literals in JSX require string concatenation for complex expressions
- Use `type` imports when importing types from modules
- Remove unused imports to prevent build warnings

### Art Deco Theme Applied
- Brass (#B57D41) primary accents
- Copper (#B87333) secondary accents
- Steel (#7A7F99) muted text
- Deco corners on cards
- Gradient lines in headers

### Build Status
✅ Build succeeds
✅ TypeScript compiles without errors
✅ Static pages generated successfully

## 2026-02-25 Task 1.7.5: Dialog Component Unit Test

### Testing Approach
- Used Jest + React Testing Library + user-event for interactions
- Mocked @radix-ui/react-dialog Portal to render inline (prevents portal-related test issues)
- All Dialog sub-components tested: Dialog, DialogTrigger, DialogContent, DialogHeader, DialogFooter, DialogTitle, DialogDescription, DialogClose

### Key Testing Patterns
1. **Portal Mocking**: Essential for Radix UI Dialog testing
   ```typescript
   jest.mock('@radix-ui/react-dialog', () => {
     const actual = jest.requireActual('@radix-ui/react-dialog');
     return {
       ...actual,
       Portal: ({ children }) => <>{children}</>,
     };
   });
   ```

2. **Accessibility Requirements**: Radix UI DialogContent requires DialogTitle for a11y
   - Tests include DialogTitle in all renders to avoid console warnings
   - Description is also recommended for screen readers

3. **Interaction Testing**: Use user-event over fireEvent for realistic interactions
   - `user.click()` for trigger and close actions
   - `waitFor()` for state transitions

4. **State Testing**:
   - Uncontrolled: `defaultOpen` prop
   - Controlled: `open` + `onOpenChange` props
   - Verified both controlled and uncontrolled modes work correctly

### Test Coverage
- ✅ Dialog renders when open
- ✅ Dialog doesn't render when closed
- ✅ DialogTrigger opens the dialog
- ✅ DialogClose closes the dialog
- ✅ DialogHeader renders correctly
- ✅ DialogTitle renders correctly
- ✅ DialogDescription renders correctly
- ✅ DialogFooter renders correctly
- ✅ onOpenChange callback works (open and close)
- ✅ Controlled state management works
- ✅ Uncontrolled defaultOpen works
- ✅ Built-in close button (X icon) present in DialogContent

### Test File Location
`web/src/components/ui/__tests__/dialog.test.tsx` (303 lines, 14 test cases)

### Test Results
- All 14 tests passed
- 0 snapshots (no snapshot testing needed)
- ~1.7s execution time

### Dependencies Used
- @testing-library/react
- @testing-library/user-event
- Jest (via npx jest)

### Testing Gotchas
1. Always include DialogTitle inside DialogContent to avoid Radix UI warnings
2. Portal mocking is required for DialogContent to render in test DOM
3. use userEvent.setup() for proper event handling cleanup
4. waitFor() needed for state transitions after async interactions

## Card Component Unit Test - Task 1.7.4 (Completed)

**Date:** 2025-02-25

### Summary
Successfully created comprehensive unit tests for the Card component and all its sub-components.

### Files Created
- `web/src/components/ui/__tests__/card.test.tsx` - 35 test cases covering all Card sub-components

### Test Coverage
All 35 tests passing:
- **Card**: 6 tests (structure, children, default classes, className merging, ref forwarding, displayName)
- **CardHeader**: 5 tests (rendering, default classes, className merging, ref forwarding, displayName)
- **CardTitle**: 5 tests (rendering, default classes, className merging, ref forwarding, displayName)
- **CardDescription**: 5 tests (rendering, default classes, className merging, ref forwarding, displayName)
- **CardContent**: 5 tests (rendering, default classes, className merging, ref forwarding, displayName)
- **CardFooter**: 5 tests (rendering, default classes, className merging, ref forwarding, displayName)
- **Nested Components**: 4 tests (complete structure, content visibility, nesting structure, custom classes at each level)

### Testing Patterns Used
- React Testing Library with Jest
- `@testing-library/jest-dom` matchers (toHaveClass, toBeInTheDocument)
- TestID strategy for element selection
- Ref forwarding verification using React.createRef()
- Component displayName verification
- Class name merging verification

### Key Findings
1. Card component uses `cn()` utility for class merging - tests verify both default and custom classes coexist
2. All sub-components properly forward refs using React.forwardRef
3. displayName is set on all components for debugging purposes
4. All components render as `<div>` elements with appropriate Tailwind CSS classes
5. The compound component pattern works correctly when nested

### Dependencies Confirmed
- Jest with next/jest configuration
- @testing-library/react for component rendering
- @testing-library/jest-dom for DOM matchers
- jest-environment-jsdom for DOM simulation
- Module path mapping `@/` → `src/` works correctly in tests


## Button Component Unit Test - Task 1.7.3

### Completed: 2026-02-25

**Test File Created:** `web/src/components/ui/__tests__/button.test.tsx`

**Test Coverage (35 tests total):**

1. **Rendering Tests (3 tests)**
   - Default props rendering
   - Custom text rendering
   - Custom className application

2. **Variant Tests (6 tests)**
   - default variant (bg-primary, text-primary-foreground)
   - destructive variant (bg-destructive, text-destructive-foreground)
   - outline variant (border, border-input, bg-background)
   - secondary variant (bg-secondary, text-secondary-foreground)
   - ghost variant (hover:bg-accent)
   - link variant (text-primary, underline-offset-4)

3. **Size Tests (4 tests)**
   - default size (h-9, px-4, py-2)
   - sm size (h-8, px-3, text-xs)
   - lg size (h-10, px-8)
   - icon size (h-9, w-9)

4. **Click Handler Tests (3 tests)**
   - Single click triggers onClick
   - Multiple clicks count correctly
   - Event object passed to handler

5. **Disabled State Tests (3 tests)**
   - Disabled attribute rendered
   - Disabled styling classes (pointer-events-none, opacity-50)
   - onClick not triggered when disabled

6. **asChild Prop Tests (4 tests)**
   - Default renders as <button>
   - Renders custom child element (e.g., <a>)
   - Applies button classes to child
   - Works with complex nested children

7. **Type Attribute Tests (3 tests)**
   - Renders as <button> element by default
   - type="submit" when specified
   - type="reset" when specified

8. **Ref Forwarding Tests (2 tests)**
   - Ref is forwarded to HTMLButtonElement
   - Focus method accessible via ref

9. **Additional Props Tests (4 tests)**
   - data-* attributes pass through
   - aria-* attributes pass through
   - id attribute pass through
   - name attribute pass through

10. **Combined Props Tests (3 tests)**
    - Multiple props work together
    - Disabled + variant combination
    - asChild + size variant combination

**Testing Setup:**
- Used Jest with Next.js configuration
- React Testing Library for DOM assertions
- jest-dom for custom matchers (toBeInTheDocument, toHaveClass, etc.)
- @testing-library/react for render, screen, fireEvent

**Key Learnings:**
- shadcn/ui Button uses @radix-ui/react-slot for asChild functionality
- class-variance-authority (cva) generates class strings but tests verify output classes
- Testing both element type and CSS classes ensures component contracts
- asChild prop requires specific testing pattern with Slot component

