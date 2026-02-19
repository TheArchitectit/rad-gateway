'use client';

import { useState } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';

interface TaskResponse {
  task: {
    id: string;
    status: string;
    sessionId: string;
    message: { content: string };
    artifacts?: Array<{ type: string; content: { text?: string } }>;
  };
}

export default function A2APage() {
  const [sessionId, setSessionId] = useState('web-session');
  const [prompt, setPrompt] = useState('Summarize gateway health in one line');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<TaskResponse | null>(null);
  const [error, setError] = useState('');

  const sendTask = async () => {
    setLoading(true);
    setError('');
    setResult(null);
    try {
      const response = await fetch('/a2a/tasks/send', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sessionId,
          message: {
            role: 'user',
            content: prompt,
          },
        }),
      });
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }
      const data = (await response.json()) as TaskResponse;
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send task');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppLayout>
      <div className="space-y-6 max-w-4xl">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">A2A Tasks</h1>
          <p className="text-gray-500">Send and inspect Agent-to-Agent tasks</p>
        </div>

        <Card className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Session ID</label>
            <input
              value={sessionId}
              onChange={(e) => setSessionId(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Prompt</label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              rows={4}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg"
            />
          </div>
          <Button onClick={sendTask} disabled={loading || !sessionId || !prompt}>
            {loading ? 'Sending...' : 'Send Task'}
          </Button>
        </Card>

        {error && (
          <Card className="p-4 border border-red-300 bg-red-50 text-red-700">{error}</Card>
        )}

        {result && (
          <Card className="p-6 space-y-2">
            <div className="text-sm text-gray-500">Task ID: {result.task.id}</div>
            <div className="text-sm text-gray-500">Status: {result.task.status}</div>
            <div className="font-medium text-gray-900">Message: {result.task.message.content}</div>
            <pre className="bg-gray-100 rounded p-3 text-sm overflow-auto">
              {JSON.stringify(result.task.artifacts ?? [], null, 2)}
            </pre>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
