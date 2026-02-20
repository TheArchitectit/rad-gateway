'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  LayoutDashboard,
  Server, 
  Key, 
  BarChart3, 
  Activity, 
  Layers,
  Zap,
  Share2,
  Shield,
  Terminal,
  FileText,
  X
} from 'lucide-react';

interface NavItem {
  label: string;
  href: string;
  icon: React.ReactNode;
  section: 'overview' | 'resources' | 'protocols' | 'analytics';
}

const navItems: NavItem[] = [
  { label: 'Dashboard', href: '/', icon: <LayoutDashboard className="h-5 w-5" />, section: 'overview' },
  { label: 'Control Rooms', href: '/control-rooms', icon: <Activity className="h-5 w-5" />, section: 'overview' },
  { label: 'Providers', href: '/providers', icon: <Server className="h-5 w-5" />, section: 'resources' },
  { label: 'API Keys', href: '/api-keys', icon: <Key className="h-5 w-5" />, section: 'resources' },
  { label: 'Projects', href: '/projects', icon: <Layers className="h-5 w-5" />, section: 'resources' },
  { label: 'A2A', href: '/a2a', icon: <Share2 className="h-5 w-5" />, section: 'protocols' },
  { label: 'OAuth', href: '/oauth', icon: <Shield className="h-5 w-5" />, section: 'protocols' },
  { label: 'MCP', href: '/mcp', icon: <Terminal className="h-5 w-5" />, section: 'protocols' },
  { label: 'Usage', href: '/usage', icon: <BarChart3 className="h-5 w-5" />, section: 'analytics' },
  { label: 'Reports', href: '/reports', icon: <FileText className="h-5 w-5" />, section: 'analytics' },
];

interface SidebarProps {
  mobileOpen: boolean;
  onCloseMobile: () => void;
}

const sectionLabels: Record<NavItem['section'], string> = {
  overview: 'Command Deck',
  resources: 'Assets',
  protocols: 'Protocols',
  analytics: 'Telemetry',
};

export function Sidebar({ mobileOpen, onCloseMobile }: SidebarProps) {
  const pathname = usePathname();
  const sections: NavItem['section'][] = ['overview', 'resources', 'protocols', 'analytics'];

  return (
    <>
      {mobileOpen && (
        <button
          type="button"
          aria-label="Close sidebar"
          onClick={onCloseMobile}
          className="fixed inset-0 z-30 bg-black/45 backdrop-blur-sm md:hidden"
        />
      )}

      <aside
        className={`fixed inset-y-0 left-0 z-40 w-72 transform border-r border-[rgba(208,173,98,0.2)] bg-[var(--surface-rail)] text-[var(--surface-panel)] shadow-[0_18px_30px_rgba(0,0,0,0.45)] transition-transform duration-300 md:translate-x-0 ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between border-b border-[rgba(208,173,98,0.2)] px-5 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-md border border-[rgba(208,173,98,0.35)] bg-gradient-to-br from-[#c89a45] via-[#9c6f2b] to-[#6e4e1f] text-[#22170e] shadow-[inset_0_1px_0_rgba(255,255,255,0.45)]">
              <Zap className="h-5 w-5" />
            </div>
            <div>
              <h1 className="text-base font-semibold uppercase tracking-[0.12em] text-[#f3e6cc]">Brass Relay</h1>
              <p className="text-xs uppercase tracking-[0.08em] text-[#bfa881]">Operations Console</p>
            </div>
          </div>
          <button
            type="button"
            aria-label="Close menu"
            onClick={onCloseMobile}
            className="rounded-md p-1.5 text-[#d4c09a] hover:bg-[rgba(255,255,255,0.08)] md:hidden"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto px-4 py-4">
          <div className="space-y-5">
            {sections.map((section) => {
              const items = navItems.filter((item) => item.section === section);
              return (
                <div key={section} className="space-y-2">
                  <p className="px-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-[#bfa881]">
                    {sectionLabels[section]}
                  </p>
                  <ul className="space-y-1">
                    {items.map((item) => {
                      const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
                      return (
                        <li key={item.href}>
                          <Link
                            href={item.href}
                            onClick={onCloseMobile}
                            className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors ${
                              isActive
                                ? 'bg-gradient-to-r from-[#c89a45] to-[#9a6e2a] text-[#1f1710] shadow-[inset_0_1px_0_rgba(255,255,255,0.35)]'
                                : 'text-[#d8c7a5] hover:bg-[rgba(255,255,255,0.08)] hover:text-[#fbf1df]'
                            }`}
                          >
                            {item.icon}
                            <span className="font-medium tracking-[0.01em]">{item.label}</span>
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                </div>
              );
            })}
          </div>
        </nav>

        <div className="border-t border-[rgba(208,173,98,0.2)] p-4">
          <div className="ui-panel-soft flex items-center gap-3 rounded-lg px-3 py-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-[#c89a45] via-[#936222] to-[#5c3f18] text-sm font-semibold text-[#241911]">
              A
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-[#2a1d12]">Admin Operator</p>
              <p className="truncate text-xs text-[#5e4a35]">admin@radgateway.io</p>
            </div>
          </div>
        </div>
      </aside>
    </>
  );
}
