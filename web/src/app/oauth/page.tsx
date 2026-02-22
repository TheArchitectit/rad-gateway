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

interface OAuthSessionResponse {
  session: {
    id: string;
    provider: string;
    status: string;
    createdAt: string;
    updatedAt: string;
    token?: {
      accessToken: string;
      refreshToken: string;
      expiresAt: string;
      tokenType: string;
    };
  };
}

interface OAuthValidateResponse {
  valid: boolean;
  metadata?: Record<string, unknown>;
  error?: string;
}

export default function OAuthPage() {
  const [provider, setProvider] = useState('openai');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<OAuthStartResponse | null>(null);
  const [sessionData, setSessionData] = useState<OAuthSessionResponse | null>(null);
  const [validation, setValidation] = useState<OAuthValidateResponse | null>(null);
  const [sessionId, setSessionId] = useState('');
  const [accessToken, setAccessToken] = useState('');
  const [error, setError] = useState('');

  const start = async () => {
    setLoading(true);
    setError('');
    setResult(null);

    try {
      const response = await fetch('/v1/oauth/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider, redirectUri: `${window.location.origin}/oauth` }),
      });
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }
      const data = (await response.json()) as OAuthStartResponse;
      setResult(data);
      setSessionId(data.sessionId);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start OAuth');
    } finally {
      setLoading(false);
    }
  };

  const loadSession = async () => {
    if (!sessionId.trim()) {
      return;
    }

    setLoading(true);
    setError('');

    try {
      const response = await fetch(`/v1/oauth/session/${encodeURIComponent(sessionId.trim())}`);
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }
      const data = (await response.json()) as OAuthSessionResponse;
      setSessionData(data);
      if (data.session?.token?.accessToken) {
        setAccessToken(data.session.token.accessToken);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load session');
    } finally {
      setLoading(false);
    }
  };

  const revokeSession = async () => {
    if (!sessionId.trim()) {
      return;
    }

    setLoading(true);
    setError('');

    try {
      const response = await fetch(`/v1/oauth/revoke/${encodeURIComponent(sessionId.trim())}`, {
        method: 'POST',
      });
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }
      await loadSession();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revoke session');
    } finally {
      setLoading(false);
    }
  };

  const validateToken = async () => {
    if (!accessToken.trim()) {
      setValidation({ valid: false, error: 'Access token required' });
      return;
    }

    setLoading(true);
    setError('');

    try {
      const response = await fetch('/v1/oauth/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider, accessToken }),
      });
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }
      const data = (await response.json()) as OAuthValidateResponse;
      setValidation(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to validate token');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-[var(--ink-900)]">OAuth Providers</h1>
          <p className="text-[var(--ink-500)]">Session lifecycle console for connected provider credentials.</p>
        </div>

        <div className="grid gap-6 xl:grid-cols-2">
          <Card title="Start OAuth Session">
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                  Provider
                </label>
                <select value={provider} onChange={(event) => setProvider(event.target.value)} className="ui-input">
                  <option value="openai">OpenAI</option>
                  <option value="openai-codex">OpenAI Codex</option>
                  <option value="anthropic">Anthropic</option>
                  <option value="github-copilot">GitHub Copilot</option>
                  <option value="gemini-cli">Gemini CLI</option>
                </select>
              </div>
              <Button onClick={() => void start()} disabled={loading}>
                {loading ? 'Starting...' : 'Start OAuth Flow'}
              </Button>

              {result && (
                <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] p-3 text-sm text-[var(--ink-700)]">
                  <p>
                    Session: <span className="font-semibold text-[var(--ink-900)]">{result.sessionId}</span>
                  </p>
                  <p>
                    State: <span className="font-semibold text-[var(--ink-900)]">{result.state}</span>
                  </p>
                  <p>
                    Status: <span className="font-semibold text-[var(--ink-900)]">{result.status}</span>
                  </p>
                  <a href={result.authUrl} target="_blank" rel="noreferrer" className="text-[var(--status-info)] underline-offset-2 hover:underline">
                    Open provider authorization URL
                  </a>
                </div>
              )}
            </div>
          </Card>

          <Card title="Session Inspector">
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                  Session ID
                </label>
                <input
                  value={sessionId}
                  onChange={(event) => setSessionId(event.target.value)}
                  placeholder="oauth session id"
                  className="ui-input"
                />
              </div>

              <div className="flex flex-wrap gap-2">
                <Button variant="secondary" onClick={() => void loadSession()} disabled={loading || !sessionId.trim()}>
                  Load Session
                </Button>
                <Button variant="danger" onClick={() => void revokeSession()} disabled={loading || !sessionId.trim()}>
                  Revoke Session
                </Button>
              </div>

              {sessionData && (
                <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] p-3 text-sm text-[var(--ink-700)]">
                  <p>
                    Provider: <span className="font-semibold text-[var(--ink-900)]">{sessionData.session.provider}</span>
                  </p>
                  <p>
                    Status: <span className="font-semibold text-[var(--ink-900)]">{sessionData.session.status}</span>
                  </p>
                  <p>
                    Updated: <span className="font-semibold text-[var(--ink-900)]">{new Date(sessionData.session.updatedAt).toLocaleString()}</span>
                  </p>
                </div>
              )}
            </div>
          </Card>
        </div>

        <Card title="Token Validation">
          <div className="space-y-4">
            <textarea
              value={accessToken}
              onChange={(event) => setAccessToken(event.target.value)}
              rows={4}
              className="ui-input font-mono text-xs"
              placeholder="Paste access token to validate"
            />
            <Button variant="secondary" onClick={() => void validateToken()} disabled={loading}>
              Validate Access Token
            </Button>

            {validation && (
              <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] p-3 text-sm text-[var(--ink-700)]">
                <p>
                  Result:{' '}
                  <span className={`font-semibold ${validation.valid ? 'text-[var(--status-normal)]' : 'text-[var(--status-critical)]'}`}>
                    {validation.valid ? 'Valid' : 'Invalid'}
                  </span>
                </p>
                {validation.error && <p className="text-[var(--status-critical)]">{validation.error}</p>}
                {validation.metadata && (
                  <pre className="mt-2 overflow-auto rounded bg-[rgba(43,32,21,0.08)] p-2 text-xs text-[var(--ink-900)]">
                    {JSON.stringify(validation.metadata, null, 2)}
                  </pre>
                )}
              </div>
            )}
          </div>
        </Card>

        {error && (
          <Card className="border border-[rgba(152,43,33,0.3)] bg-[rgba(152,43,33,0.08)]">
            <p className="text-sm text-[var(--status-critical)]">{error}</p>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
