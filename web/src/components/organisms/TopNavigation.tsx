'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Bell, ChevronRight, Home, Menu, Search } from 'lucide-react';
import { Avatar } from '../atoms/Avatar';

function Breadcrumb() {
  const pathname = usePathname();
  const segments = pathname.split('/').filter(Boolean);

  if (segments.length === 0) {
    return (
      <div className="flex items-center gap-2 text-sm text-[var(--ink-500)]">
        <Home className="w-4 h-4" />
        <span>Dashboard</span>
      </div>
    );
  }

  return (
    <nav className="flex items-center gap-2 text-sm">
      <Link href="/" className="text-[var(--ink-500)] hover:text-[var(--ink-700)]">
        <Home className="w-4 h-4" />
      </Link>
      {segments.map((segment, index) => {
        const href = '/' + segments.slice(0, index + 1).join('/');
        const isLast = index === segments.length - 1;
        const label = segment.charAt(0).toUpperCase() + segment.slice(1);

        return (
          <React.Fragment key={href}>
            <ChevronRight className="w-4 h-4 text-[var(--ink-500)]" />
            {isLast ? (
              <span className="font-medium text-[var(--ink-900)]">{label.replace('-', ' ')}</span>
            ) : (
              <Link href={href} className="capitalize text-[var(--ink-500)] hover:text-[var(--ink-700)]">
                {label.replace('-', ' ')}
              </Link>
            )}
          </React.Fragment>
        );
      })}
    </nav>
  );
}

interface TopNavigationProps {
  onMenuToggle: () => void;
}

export function TopNavigation({ onMenuToggle }: TopNavigationProps) {
  return (
    <header className="sticky top-0 z-20 flex h-16 items-center justify-between border-b border-[var(--line-strong)] ui-panel px-4 md:px-6">
      <div className="flex items-center gap-3 md:gap-5">
        <button
          type="button"
          onClick={onMenuToggle}
          aria-label="Toggle navigation"
          className="rounded-md p-2 text-[var(--ink-700)] hover:bg-[rgba(43,32,21,0.08)] md:hidden"
        >
          <Menu className="h-5 w-5" />
        </button>
        <Breadcrumb />
      </div>

      <div className="hidden items-center gap-2 rounded-lg border border-[var(--line-soft)] bg-[rgba(255,255,255,0.4)] px-3 py-1.5 lg:flex">
        <Search className="h-4 w-4 text-[var(--ink-500)]" />
        <input
          type="search"
          placeholder="Search objects"
          className="w-44 bg-transparent text-sm text-[var(--ink-900)] placeholder:text-[var(--ink-500)] focus:outline-none"
        />
      </div>

      <div className="flex items-center gap-2 md:gap-4">
        <select className="hidden rounded-md border border-[var(--line-soft)] bg-[rgba(255,255,255,0.4)] px-2.5 py-1.5 text-sm text-[var(--ink-700)] md:block">
          <option>Production Platform</option>
          <option>Staging Platform</option>
          <option>Cost Sentinel Room</option>
        </select>

        <button
          type="button"
          className="relative rounded-md p-2 text-[var(--ink-700)] hover:bg-[rgba(43,32,21,0.08)]"
        >
          <Bell className="h-5 w-5" />
          <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-[var(--status-critical)]" />
        </button>

        <div className="hidden h-6 w-px bg-[var(--line-soft)] md:block" />

        <div className="flex items-center gap-3">
          <Avatar name="Admin User" size="sm" />
          <div className="hidden md:block">
            <p className="text-sm font-medium text-[var(--ink-900)]">Admin Operator</p>
            <p className="text-xs uppercase tracking-[0.08em] text-[var(--ink-500)]">Master Console</p>
          </div>
        </div>
      </div>
    </header>
  );
}
