'use client';

import { useState } from 'react';
import { Check, Copy, Eye, EyeOff, Key, AlertTriangle, Shield, ShieldCheck, X } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';

interface KeyRevealProps {
  isOpen: boolean;
  onClose: () => void;
  apiKeyName: string;
  apiKeySecret: string;
  apiKeyPreview: string;
}

export function KeyReveal({
  isOpen,
  onClose,
  apiKeyName,
  apiKeySecret,
  apiKeyPreview,
}: KeyRevealProps) {
  const [copied, setCopied] = useState(false);
  const [showKey, setShowKey] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(apiKeySecret);
      setCopied(true);
      setTimeout(() => setCopied(false), 3000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const maskedKey = apiKeySecret.slice(0, 8) + '•'.repeat(apiKeySecret.length - 16) + apiKeySecret.slice(-8);

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[550px] overflow-hidden">
        {/* Art Deco Header */}
        <div className="relative -mx-6 -mt-6 px-6 pt-6 pb-4 bg-gradient-to-br from-[hsl(33,43%,15%)] via-[hsl(33,43%,25%)] to-[hsl(33,43%,35%)] border-b-2 border-[hsl(33,43%,48%)]">
          {/* Decorative corner elements */}
          <div className="absolute top-3 left-3 w-8 h-8 border-l-2 border-t-2 border-[hsl(33,43%,62%)]/50" />
          <div className="absolute top-3 right-3 w-8 h-8 border-r-2 border-t-2 border-[hsl(33,43%,62%)]/50" />
          
          {/* Geometric pattern overlay */}
          <div 
            className="absolute inset-0 opacity-10 pointer-events-none"
            style={{
              backgroundImage: `
                repeating-linear-gradient(
                  45deg,
                  transparent,
                  transparent 10px,
                  rgba(177, 133, 50, 0.1) 10px,
                  rgba(177, 133, 50, 0.1) 20px
                ),
                repeating-linear-gradient(
                  -45deg,
                  transparent,
                  transparent 10px,
                  rgba(177, 133, 50, 0.05) 10px,
                  rgba(177, 133, 50, 0.05) 20px
                )
              `
            }}
          />

          <DialogHeader className="relative z-10">
            <div className="flex items-center justify-center gap-3">
              <div className="relative">
                <div className="absolute -inset-1 bg-[hsl(33,43%,48%)]/30 rounded-full blur-sm" />
                <div className="relative p-3 rounded-full bg-[hsl(33,43%,48%)]/20 border border-[hsl(33,43%,62%)]/50">
                  <Key className="h-6 w-6 text-[hsl(33,43%,72%)]" />
                </div>
              </div>
              <div>
                <DialogTitle className="text-xl font-bold text-[hsl(33,43%,95%)] font-display tracking-wide">
                  API Key Created
                </DialogTitle>
                <DialogDescription className="text-[hsl(33,43%,75%)] text-sm mt-1">
                  Your new key has been generated successfully
                </DialogDescription>
              </div>
            </div>
          </DialogHeader>

          {/* Decorative line */}
          <div className="mt-4 flex items-center justify-center gap-2">
            <div className="h-px flex-1 bg-gradient-to-r from-transparent via-[hsl(33,43%,62%)]/50 to-transparent" />
            <div className="w-2 h-2 rotate-45 bg-[hsl(33,43%,48%)]" />
            <div className="h-px flex-1 bg-gradient-to-r from-transparent via-[hsl(33,43%,62%)]/50 to-transparent" />
          </div>
        </div>

        {/* Warning Banner */}
        <div className="relative -mx-6 px-6 py-3 bg-gradient-to-r from-amber-900/40 via-amber-800/30 to-amber-900/40 border-y border-amber-600/30">
          <div className="flex items-center gap-3">
            <div className="p-1.5 rounded-full bg-amber-500/20 border border-amber-500/30">
              <AlertTriangle className="h-4 w-4 text-amber-400" />
            </div>
            <div className="flex-1">
              <p className="text-amber-200 text-sm font-medium">
                This key will only be shown once
              </p>
              <p className="text-amber-200/70 text-xs">
                Copy it now and store it securely. You won&apos;t be able to see it again.
              </p>
            </div>
          </div>
        </div>

        {/* Key Details */}
        <div className="space-y-6 py-4">
          {/* Key Name */}
          <div className="space-y-2">
            <label className="text-sm font-medium text-[hsl(var(--muted-foreground))]">
              Key Name
            </label>
            <div className="flex items-center gap-2 p-3 rounded-lg bg-[hsl(var(--muted))]/20 border border-[hsl(var(--border))]">
              <ShieldCheck className="h-4 w-4 text-[hsl(var(--primary))]" />
              <span className="font-medium">{apiKeyName}</span>
              <Badge variant="secondary" className="ml-auto text-xs">Active</Badge>
            </div>
          </div>

          {/* The Secret Key */}
          <div className="space-y-2">
            <label className="text-sm font-medium text-[hsl(var(--muted-foreground))]">
              Secret Key
            </label>
            <div className="relative">
              {/* Art Deco frame around key */}
              <div className="absolute -inset-1 bg-gradient-to-br from-[hsl(33,43%,48%)]/20 via-transparent to-[hsl(33,43%,48%)]/20 rounded-lg blur-sm" />
              
              <div className="relative flex items-center gap-2 p-4 rounded-lg bg-gradient-to-br from-[hsl(33,43%,15%)] to-[hsl(33,43%,10%)] border-2 border-[hsl(33,43%,48%)]/50">
                {/* Decorative corners */}
                <div className="absolute top-0 left-0 w-3 h-3 border-t-2 border-l-2 border-[hsl(33,43%,62%)]" />
                <div className="absolute top-0 right-0 w-3 h-3 border-t-2 border-r-2 border-[hsl(33,43%,62%)]" />
                <div className="absolute bottom-0 left-0 w-3 h-3 border-b-2 border-l-2 border-[hsl(33,43%,62%)]" />
                <div className="absolute bottom-0 right-0 w-3 h-3 border-b-2 border-r-2 border-[hsl(33,43%,62%)]" />
                
                <code className="flex-1 font-mono text-sm tracking-wide text-[hsl(33,43%,85%)] break-all">
                  {showKey ? apiKeySecret : maskedKey}
                </code>
                
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setShowKey(!showKey)}
                    className="h-8 w-8 text-[hsl(33,43%,72%)] hover:text-[hsl(33,43%,95%)] hover:bg-[hsl(33,43%,48%)]/20"
                  >
                    {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleCopy}
                    className={`h-8 w-8 transition-all duration-300 ${
                      copied 
                        ? 'text-green-400 bg-green-500/20 hover:bg-green-500/30' 
                        : 'text-[hsl(33,43%,72%)] hover:text-[hsl(33,43%,95%)] hover:bg-[hsl(33,43%,48%)]/20'
                    }`}
                  >
                    {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                  </Button>
                </div>
              </div>
            </div>
          </div>

          {/* Key Preview */}
          <div className="space-y-2">
            <label className="text-sm font-medium text-[hsl(var(--muted-foreground))]">
              Key Preview
            </label>
            <div className="flex items-center gap-2 p-3 rounded-lg bg-[hsl(var(--muted))]/10 border border-[hsl(var(--border))]">
              <code className="font-mono text-sm text-[hsl(var(--muted-foreground))]">
                {apiKeyPreview}
              </code>
            </div>
            <p className="text-xs text-[hsl(var(--muted-foreground))]">
              This preview is shown in the API keys list
            </p>
          </div>

          {/* Security Tips */}
          <div className="p-4 rounded-lg bg-gradient-to-br from-[hsl(220,15%,15%)] to-[hsl(220,15%,12%)] border border-[hsl(220,15%,25%)]">
            <div className="flex items-start gap-3">
              <div className="p-1.5 rounded-full bg-[hsl(33,43%,48%)]/20">
                <Shield className="h-4 w-4 text-[hsl(33,43%,62%)]" />
              </div>
              <div className="flex-1 space-y-2">
                <h4 className="font-medium text-sm text-[hsl(var(--foreground))]">
                  Security Best Practices
                </h4>
                <ul className="space-y-1.5 text-xs text-[hsl(var(--muted-foreground))]">
                  <li className="flex items-start gap-2">
                    <span className="text-[hsl(33,43%,48%)] mt-0.5">◆</span>
                    Store this key in a secure location like a password manager
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-[hsl(33,43%,48%)] mt-0.5">◆</span>
                    Never commit API keys to version control or share them in public
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-[hsl(33,43%,48%)] mt-0.5">◆</span>
                    Rotate keys regularly and revoke unused keys promptly
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <DialogFooter className="flex flex-col sm:flex-row gap-3 pt-2 border-t border-[hsl(var(--border))]">
          <Button
            variant="outline"
            onClick={onClose}
            className="w-full sm:w-auto order-2 sm:order-1"
          >
            <X className="mr-2 h-4 w-4" />
            Close
          </Button>
          <Button
            onClick={handleCopy}
            className={`w-full sm:w-auto order-1 sm:order-2 transition-all duration-300 ${
              copied 
                ? 'bg-green-600 hover:bg-green-700 text-white' 
                : ''
            }`}
          >
            {copied ? (
              <>
                <Check className="mr-2 h-4 w-4" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="mr-2 h-4 w-4" />
                Copy Key to Clipboard
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
