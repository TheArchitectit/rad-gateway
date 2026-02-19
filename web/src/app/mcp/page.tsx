'use client';

import { useState } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';

export default function MCPPage() {
  const [tool, setTool] = useState('list_models');
  const [input, setInput] = useState('{"limit": 5}');
  const [loading, setLoading] = useState(false);
  const [output, setOutput] = useState('');
  const [error, setError] = useState('');

  const invoke = async () => {
    setLoading(true);
    setError('');
    setOutput('');
    try {
      const parsed = input.trim() ? JSON.parse(input) : {};
      const response = await fetch('/mcp/v1/stdio', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tool, input: parsed }),
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
      <div className="space-y-6 max-w-4xl">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">MCP Console</h1>
          <p className="text-gray-500">Call MCP proxy tools over HTTP</p>
        </div>

        <Card className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Tool</label>
            <input
              value={tool}
              onChange={(e) => setTool(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Input JSON</label>
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              rows={6}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg font-mono text-sm"
            />
          </div>
          <Button onClick={invoke} disabled={loading}>{loading ? 'Invoking...' : 'Invoke MCP Tool'}</Button>
        </Card>

        {error && <Card className="p-4 border border-red-300 bg-red-50 text-red-700">{error}</Card>}

        {output && (
          <Card className="p-4">
            <pre className="bg-gray-100 rounded p-3 text-sm overflow-auto">{output}</pre>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
