import React from 'react';
import { Sidebar } from '../organisms/Sidebar';
import { TopNavigation } from '../organisms/TopNavigation';

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  return (
    <div className="min-h-screen bg-gray-50">
      <Sidebar />
      
      <div className="ml-64">
        <TopNavigation />
        
        <main className="p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
