import { useEffect, useState } from 'react'

const DEFAULT_AGENT_BASE_URL = '/agent'

type AgentEvidence = { field: string; value: string | number | boolean | null; timestamp?: string | null }

type AgentDiagnostics = {
  terminal: boolean
  stuck: boolean
  blockers: string[]
  recommended_actions: string[]
}

type AgentTimeline = { event: string; timestamp: string }

type AgentData = {
  summary: string
  evidence: AgentEvidence[]
  diagnostics: AgentDiagnostics
  timeline: AgentTimeline[]
  next_steps: string[]
}

type AgentOk = { ok: true; request_id: string; data: AgentData }

type AgentError = { ok: false; request_id: string; error: { code: string; message: string } }

type AgentResult = AgentOk | AgentError

type AgentPanelProps = {
  prefillStudyId?: string
  prefillStudyUid?: string
}

export function AgentPanel({ prefillStudyId, prefillStudyUid }: AgentPanelProps) {
  const baseUrl = import.meta.env.VITE_AGENT_BASE_URL || DEFAULT_AGENT_BASE_URL
  const [studyId, setStudyId] = useState('')
  const [studyUid, setStudyUid] = useState('')
  const [question, setQuestion] = useState('')
  const [includeNextSteps, setIncludeNextSteps] = useState(true)
  const [agentApiKey, setAgentApiKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<AgentResult | null>(null)

  useEffect(() => {
    if (prefillStudyId || prefillStudyUid) {
      setStudyId(prefillStudyId || '')
      setStudyUid(prefillStudyUid || '')
    }
  }, [prefillStudyId, prefillStudyUid])

  async function submit() {
    setError(null)
    setResult(null)

    if (!studyId.trim() && !studyUid.trim()) {
      setError('Provide a study ID or StudyInstanceUID.')
      return
    }

    const payload: Record<string, unknown> = {
      question: question.trim() || undefined,
      study_id: studyId.trim() || undefined,
      study_instance_uid: studyUid.trim() || undefined,
      include_next_steps: includeNextSteps
    }

    setLoading(true)
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (agentApiKey.trim()) headers.Authorization = `Bearer ${agentApiKey.trim()}`

      const response = await fetch(`${baseUrl}/ask`, {
        method: 'POST',
        headers,
        body: JSON.stringify(payload)
      })

      const contentType = response.headers.get('content-type') ?? ''
      if (!contentType.includes('application/json')) {
        throw new Error(`Agent service unavailable (HTTP ${response.status}) — is the MCP server running?`)
      }

      const data = await response.json()
      setResult(data as AgentResult)
      if (!response.ok) {
        const msg = (data as AgentError)?.error?.message || 'Agent request failed'
        setError(msg)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Agent request failed')
    } finally {
      setLoading(false)
    }
  }

  function resetForm() {
    setStudyId('')
    setStudyUid('')
    setQuestion('')
    setIncludeNextSteps(true)
    setAgentApiKey('')
    setError(null)
    setResult(null)
  }

  const data = result && result.ok ? result.data : null

  return (
    <div className="agent-panel">
      <div className="agent-panel__header">
        <div>
          <h2>AEGIS Agent</h2>
          <p>Read-only study status and diagnostics assistant</p>
        </div>
      </div>

      <div className="agent-form">
        <div className="form-row">
          <input
            className="form-input"
            type="text"
            placeholder="Study UUID"
            value={studyId}
            onChange={(e) => setStudyId(e.target.value)}
          />
          <input
            className="form-input"
            type="text"
            placeholder="StudyInstanceUID"
            value={studyUid}
            onChange={(e) => setStudyUid(e.target.value)}
          />
        </div>
        <div className="form-row">
          <input
            className="form-input"
            type="text"
            placeholder="Question (optional)"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
          />
        </div>
        <div className="form-row">
          <input
            className="form-input"
            type="password"
            placeholder="Agent API key (optional)"
            value={agentApiKey}
            onChange={(e) => setAgentApiKey(e.target.value)}
          />
        </div>
        <label className="form-checkbox">
          <input
            type="checkbox"
            checked={includeNextSteps}
            onChange={(e) => setIncludeNextSteps(e.target.checked)}
          />
          Include next steps
        </label>
        <div className="form-row form-row--actions">
          <button type="button" className="btn-primary" onClick={submit} disabled={loading}>
            {loading ? 'Running…' : 'Ask agent'}
          </button>
          <button type="button" className="btn-secondary" onClick={resetForm} disabled={loading}>
            Clear
          </button>
        </div>
        {error && <div className="form-error">{error}</div>}
      </div>

      {data && (
        <div className="agent-output">
          <div className="agent-section">
            <h3>Summary</h3>
            <p>{data.summary}</p>
          </div>

          <div className="agent-grid">
            <div className="agent-section">
              <h3>Evidence</h3>
              {data.evidence.length === 0 && <p className="agent-muted">No evidence entries returned.</p>}
              {data.evidence.map((item, idx) => (
                <div key={`${item.field}-${idx}`} className="agent-kv">
                  <span className="agent-kv__field">{item.field}</span>
                  <span className="agent-kv__value">{String(item.value)}</span>
                  {item.timestamp && <span className="agent-kv__meta">{item.timestamp}</span>}
                </div>
              ))}
            </div>

            <div className="agent-section">
              <h3>Diagnostics</h3>
              <div className="agent-kv">
                <span className="agent-kv__field">Terminal</span>
                <span className={`status-badge status-badge--${data.diagnostics.terminal ? 'clean' : 'failed'}`}>
                  {data.diagnostics.terminal ? 'yes' : 'no'}
                </span>
              </div>
              <div className="agent-kv">
                <span className="agent-kv__field">Stuck</span>
                <span className={`status-badge status-badge--${data.diagnostics.stuck ? 'failed' : 'clean'}`}>
                  {data.diagnostics.stuck ? 'yes' : 'no'}
                </span>
              </div>
              {data.diagnostics.blockers.length === 0 && <p className="agent-muted">No blockers listed.</p>}
              {data.diagnostics.blockers.map((blocker, idx) => (
                <div key={`blocker-${idx}`} className="agent-pill agent-pill--blocker">{blocker}</div>
              ))}
            </div>
          </div>

          <div className="agent-grid">
            <div className="agent-section">
              <h3>Timeline</h3>
              {data.timeline.length === 0 && <p className="agent-muted">No recent audit events.</p>}
              {data.timeline.map((entry, idx) => (
                <div key={`timeline-${idx}`} className="agent-kv">
                  <span className="agent-kv__field">{entry.event}</span>
                  <span className="agent-kv__meta">{entry.timestamp}</span>
                </div>
              ))}
            </div>

            <div className="agent-section">
              <h3>Next steps</h3>
              {data.next_steps.length === 0 && <p className="agent-muted">No next steps requested.</p>}
              {data.next_steps.map((step, idx) => (
                <div key={`step-${idx}`} className="agent-pill agent-pill--step">{step}</div>
              ))}
            </div>
          </div>

          <details className="agent-raw">
            <summary>Raw response</summary>
            <pre>{JSON.stringify(data, null, 2)}</pre>
          </details>
        </div>
      )}
    </div>
  )
}
