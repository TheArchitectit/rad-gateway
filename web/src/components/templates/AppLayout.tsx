"use client";

import React from 'react';
import { useState } from 'react';
import { Sidebar } from '../organisms/Sidebar';
import { TopNavigation } from '../organisms/TopNavigation';

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  return (
    <div className="min-h-screen">
      <Sidebar
        mobileOpen={mobileNavOpen}
        onCloseMobile={() => setMobileNavOpen(false)}
      />

      <div className="md:ml-72">
        <TopNavigation onMenuToggle={() => setMobileNavOpen((current) => !current)} />

        <main className="p-4 md:p-6 lg:p-8">
          {children}
        </main>
      </div>
    </div>
  );
}
