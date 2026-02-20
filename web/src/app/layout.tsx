import type { Metadata } from 'next';
import { Cinzel, Source_Sans_3 } from 'next/font/google';
import './globals.css';
import { QueryProvider } from '@/queries/QueryProvider';

const displayFont = Cinzel({
  subsets: ['latin'],
  variable: '--font-display',
  weight: ['500', '700'],
});

const bodyFont = Source_Sans_3({
  subsets: ['latin'],
  variable: '--font-body',
  weight: ['400', '500', '600', '700'],
});

export const metadata: Metadata = {
  title: 'RAD Gateway Admin',
  description: 'AI API Gateway Management Console',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={`${displayFont.variable} ${bodyFont.variable}`}>
        <QueryProvider>
          {children}
        </QueryProvider>
      </body>
    </html>
  );
}
