'use client';

import { useEffect, useMemo, useState } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';

interface MCPTool {
  name: string;
  description: string;
}

interface MCPHealth {
  status: string;
  service: string;
  executor: boolean;
  timestamp: string;
}

export default function MCPPage() {
  const [tool, setTool] = useState('echo');
  const [session, setSession] = useState('ui-control-room');
  const [input, setInput] = useState('{\n  "content": "Inspect provider health and summarize anomalies."\n}');
  const [loading, setLoading] = useState(false);
  const [output, setOutput] = useState('');
  const [error, setError] = useState('');

  const [tools, setTools] = useState<MCPTool[]>([]);
  const [health, setHealth] = useState<MCPHealth | null>(null);

  useEffect(() => {
    const loadMetadata = async () => {
      try {
        const [toolsResponse, healthResponse] = await Promise.all([
          fetch('/mcp/v1/tools/list'),
          fetch('/mcp/v1/health'),
        ]);

        if (toolsResponse.ok) {
          const toolsJson = await toolsResponse.json();
          setTools((toolsJson.tools || []) as MCPTool[]);
          if (Array.isArray(toolsJson.tools) && toolsJson.tools.length > 0) {
            setTool((current) => current || toolsJson.tools[0].name || 'echo');
          }
        }

        if (healthResponse.ok) {
          const healthJson = (await healthResponse.json()) as MCPHealth;
          setHealth(healthJson);
        }
      } catch {
        setTools([]);
      }
    };

    void loadMetadata();
  }, []);

  const selectedToolDescription = useMemo(
    () => tools.find((candidate) => candidate.name === tool)?.description || '',
    [tool, tools]
  );

  const invoke = async () => {
    setLoading(true);
    setError('');
    setOutput('');

    try {
      const parsed = input.trim() ? JSON.parse(input) : {};
      const response = await fetch('/mcp/v1/tools/invoke', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tool, input: parsed, session }),
      });

      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }

      const data = await response.json();
      setOutput(JSON.stringify(data, null, 2));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invocation failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-[var(--ink-900)]">MCP Console</h1>
          <p className="text-[var(--ink-500)]">Tool bridge control panel with live invocation and health visibility.</p>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          <Card title="Tool Invocation" className="lg:col-span-2">
            <div className="space-y-4">
              <div className="grid gap-3 md:grid-cols-2">
                <div>
                  <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                    Tool
                  </label>
                  <select value={tool} onChange={(event) => setTool(event.target.value)} className="ui-input">
                    {(tools.length > 0 ? tools : [{ name: 'echo', description: 'Echo input' }]).map((candidate) => (
                      <option key={candidate.name} value={candidate.name}>
                        {candidate.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                    Session
                  </label>
                  <input
                    value={session}
                    onChange={(event) => setSession(event.target.value)}
                    className="ui-input"
                  />
                </div>
              </div>

              <div>
                <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                  Input JSON
                </label>
                <textarea
                  value={input}
                  onChange={(event) => setInput(event.target.value)}
                  rows={8}
                  className="ui-input font-mono text-sm"
                />
              </div>

              {selectedToolDescription && (
                <p className="text-sm text-[var(--ink-500)]">{selectedToolDescription}</p>
              )}

              <Button onClick={() => void invoke()} disabled={loading || !tool}>
                {loading ? 'Invoking...' : 'Invoke Tool'}
              </Button>
            </div>
          </Card>

          <Card title="Bridge Health">
            <div className="space-y-2 text-sm text-[var(--ink-700)]">
              <p>
                Status: <span className="font-semibold text-[var(--ink-900)]">{health?.status || 'unknown'}</span>
              </p>
              <p>
                Service: <span className="font-semibold text-[var(--ink-900)]">{health?.service || 'mcp'}</span>
              </p>
              <p>
                Executor: <span className="font-semibold text-[var(--ink-900)]">{health?.executor ? 'gateway-backed' : 'builtin only'}</span>
              </p>
              {health?.timestamp && (
                <p className="text-xs text-[var(--ink-500)]">Updated {new Date(health.timestamp).toLocaleString()}</p>
              )}
            </div>
          </Card>
        </div>

        {error && (
          <Card className="border border-[rgba(152,43,33,0.3)] bg-[rgba(152,43,33,0.08)]">
            <p className="text-sm text-[var(--status-critical)]">{error}</p>
          </Card>
        )}

        {output && (
          <Card title="Invocation Output">
            <pre className="overflow-auto rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] p-4 text-sm text-[var(--ink-900)]">
              {output}
            </pre>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
