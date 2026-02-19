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
  FileText
} from 'lucide-react';

interface NavItem {
  label: string;
  href: string;
  icon: React.ReactNode;
}

const navItems: NavItem[] = [
  { label: 'Dashboard', href: '/', icon: <LayoutDashboard className="w-5 h-5" /> },
  { label: 'Control Rooms', href: '/control-rooms', icon: <Activity className="w-5 h-5" /> },
  { label: 'Providers', href: '/providers', icon: <Server className="w-5 h-5" /> },
  { label: 'API Keys', href: '/api-keys', icon: <Key className="w-5 h-5" /> },
  { label: 'Projects', href: '/projects', icon: <Layers className="w-5 h-5" /> },
  { label: 'Usage', href: '/usage', icon: <BarChart3 className="w-5 h-5" /> },
  { label: 'A2A', href: '/a2a', icon: <Share2 className="w-5 h-5" /> },
  { label: 'OAuth', href: '/oauth', icon: <Shield className="w-5 h-5" /> },
  { label: 'MCP', href: '/mcp', icon: <Terminal className="w-5 h-5" /> },
  { label: 'Reports', href: '/reports', icon: <FileText className="w-5 h-5" /> },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 bg-gray-900 text-white h-screen flex flex-col fixed left-0 top-0">
      <div className="p-6 border-b border-gray-800">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-blue-500 rounded-lg flex items-center justify-center">
            <Zap className="w-5 h-5 text-white" />
          </div>
          <div>
            <h1 className="text-lg font-bold">RAD Gateway</h1>
            <p className="text-xs text-gray-400">Admin Console</p>
          </div>
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto py-4">
        <ul className="space-y-1 px-3">
          {navItems.map((item) => {
            const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                    isActive
                      ? 'bg-blue-600 text-white'
                      : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                  }`}
                >
                  {item.icon}
                  <span className="text-sm font-medium">{item.label}</span>
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      <div className="p-4 border-t border-gray-800">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-500 rounded-full flex items-center justify-center text-sm font-medium">
            A
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium truncate">Admin User</p>
            <p className="text-xs text-gray-400 truncate">admin@radgateway.io</p>
          </div>
        </div>
      </div>
    </aside>
  );
}
