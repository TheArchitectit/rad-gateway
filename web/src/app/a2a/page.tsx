'use client';

import { useState } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';

interface TaskArtifact {
  type: string;
  name?: string;
  description?: string;
  content?: unknown;
}

interface TaskPayload {
  id: string;
  status: string;
  sessionId: string;
  message: { role: string; content: string };
  createdAt?: string;
  updatedAt?: string;
  artifacts?: TaskArtifact[];
}

interface TaskResponse {
  task: TaskPayload;
}

export default function A2APage() {
  const [sessionId, setSessionId] = useState('ui-session-control-room');
  const [prompt, setPrompt] = useState('Summarize provider health anomalies in one sentence.');
  const [model, setModel] = useState('gpt-4o-mini');
  const [taskId, setTaskId] = useState('');

  const [loading, setLoading] = useState(false);
  const [taskData, setTaskData] = useState<TaskPayload | null>(null);
  const [events, setEvents] = useState<string[]>([]);
  const [error, setError] = useState('');

  const pushEvent = (event: string) => {
    setEvents((previous) => [`${new Date().toLocaleTimeString()} · ${event}`, ...previous].slice(0, 10));
  };

  const sendTask = async () => {
    setLoading(true);
    setError('');

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
          metadata: JSON.stringify({ model, api_type: 'chat' }),
        }),
      });

      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }

      const data = (await response.json()) as TaskResponse;
      setTaskData(data.task);
      setTaskId(data.task.id);
      pushEvent(`Task ${data.task.id} submitted and completed with status ${data.task.status}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send task');
      pushEvent('Task submission failed.');
    } finally {
      setLoading(false);
    }
  };

  const fetchTask = async () => {
    if (!taskId.trim()) {
      return;
    }

    setLoading(true);
    setError('');

    try {
      const response = await fetch(`/a2a/tasks/${encodeURIComponent(taskId.trim())}`);
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }

      const data = (await response.json()) as TaskResponse;
      setTaskData(data.task);
      pushEvent(`Task ${data.task.id} polled with status ${data.task.status}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load task');
      pushEvent('Task lookup failed.');
    } finally {
      setLoading(false);
    }
  };

  const cancelTask = async () => {
    if (!taskId.trim()) {
      return;
    }

    setLoading(true);
    setError('');

    try {
      const response = await fetch(`/a2a/tasks/${encodeURIComponent(taskId.trim())}/cancel`, {
        method: 'POST',
      });

      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }

      const data = (await response.json()) as TaskResponse;
      setTaskData(data.task);
      pushEvent(`Cancel request accepted for task ${data.task.id}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel task');
      pushEvent('Task cancel failed.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-[var(--ink-900)]">A2A Task Console</h1>
          <p className="text-[var(--ink-500)]">Lifecycle operations for send, query, and cancellation workflows.</p>
        </div>

        <div className="grid gap-6 xl:grid-cols-2">
          <Card title="Submit Task">
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                  Session ID
                </label>
                <input value={sessionId} onChange={(event) => setSessionId(event.target.value)} className="ui-input" />
              </div>

              <div>
                <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                  Model
                </label>
                <input value={model} onChange={(event) => setModel(event.target.value)} className="ui-input" />
              </div>

              <div>
                <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                  Prompt
                </label>
                <textarea
                  value={prompt}
                  onChange={(event) => setPrompt(event.target.value)}
                  rows={4}
                  className="ui-input"
                />
              </div>

              <Button onClick={() => void sendTask()} disabled={loading || !sessionId.trim() || !prompt.trim()}>
                {loading ? 'Submitting...' : 'Send Task'}
              </Button>
            </div>
          </Card>

          <Card title="Task Operations">
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                  Task ID
                </label>
                <input value={taskId} onChange={(event) => setTaskId(event.target.value)} className="ui-input" />
              </div>

              <div className="flex flex-wrap gap-2">
                <Button variant="secondary" onClick={() => void fetchTask()} disabled={loading || !taskId.trim()}>
                  Get Task
                </Button>
                <Button variant="danger" onClick={() => void cancelTask()} disabled={loading || !taskId.trim()}>
                  Cancel Task
                </Button>
              </div>

              <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] p-3 text-sm text-[var(--ink-700)]">
                <p>Track status transitions: submitted → working → completed/failed.</p>
                <p className="mt-1 text-xs text-[var(--ink-500)]">Use poll + cancel to validate runtime behavior.</p>
              </div>
            </div>
          </Card>
        </div>

        {error && (
          <Card className="border border-[rgba(152,43,33,0.3)] bg-[rgba(152,43,33,0.08)]">
            <p className="text-sm text-[var(--status-critical)]">{error}</p>
          </Card>
        )}

        {taskData && (
          <Card title="Task Snapshot">
            <div className="space-y-2 text-sm text-[var(--ink-700)]">
              <p>
                ID: <span className="font-semibold text-[var(--ink-900)]">{taskData.id}</span>
              </p>
              <p>
                Status: <span className="font-semibold text-[var(--ink-900)]">{taskData.status}</span>
              </p>
              <p>
                Session: <span className="font-semibold text-[var(--ink-900)]">{taskData.sessionId}</span>
              </p>
              <p>
                Message: <span className="font-semibold text-[var(--ink-900)]">{taskData.message.content}</span>
              </p>
              <pre className="mt-2 overflow-auto rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] p-3 text-xs text-[var(--ink-900)]">
                {JSON.stringify(taskData.artifacts || [], null, 2)}
              </pre>
            </div>
          </Card>
        )}

        <Card title="Event Timeline">
          <div className="space-y-2">
            {events.length === 0 && <p className="text-sm text-[var(--ink-500)]">No events recorded yet.</p>}
            {events.map((event) => (
              <div key={event} className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] px-3 py-2 text-sm text-[var(--ink-700)]">
                {event}
              </div>
            ))}
          </div>
        </Card>
      </div>
    </AppLayout>
  );
}
