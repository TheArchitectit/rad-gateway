'use client';

import { useState } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';

interface OAuthStartResponse {
  sessionId: string;
  state: string;
  authUrl: string;
  status: string;
}

export default function OAuthPage() {
  const [provider, setProvider] = useState('github-copilot');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<OAuthStartResponse | null>(null);
  const [error, setError] = useState('');

  const start = async () => {
    setLoading(true);
    setError('');
    setResult(null);
    try {
      const response = await fetch('/v1/oauth/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider, redirectUri: window.location.origin + '/oauth' }),
      });
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }
      const data = (await response.json()) as OAuthStartResponse;
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start OAuth');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppLayout>
      <div className="space-y-6 max-w-4xl">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">OAuth Providers</h1>
          <p className="text-gray-500">Connect external OAuth-backed model providers</p>
        </div>

        <Card className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Provider</label>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg"
            >
              <option value="github-copilot">GitHub Copilot</option>
              <option value="anthropic">Anthropic</option>
              <option value="gemini-cli">Gemini CLI</option>
              <option value="openai-codex">OpenAI Codex</option>
              <option value="openai">OpenAI</option>
            </select>
          </div>
          <Button onClick={start} disabled={loading}>
            {loading ? 'Starting...' : 'Start OAuth Flow'}
          </Button>
        </Card>

        {error && <Card className="p-4 border border-red-300 bg-red-50 text-red-700">{error}</Card>}

        {result && (
          <Card className="p-6 space-y-2">
            <div className="text-sm text-gray-500">Session ID: {result.sessionId}</div>
            <div className="text-sm text-gray-500">State: {result.state}</div>
            <div className="text-sm text-gray-500">Status: {result.status}</div>
            <a href={result.authUrl} target="_blank" rel="noreferrer" className="text-blue-600 hover:underline">
              Open Provider Authorization URL
            </a>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
