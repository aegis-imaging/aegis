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

const FALLBACK_MODELS = [
  { value: 'google/gemini-2.5-pro',        label: 'Gemini 2.5 Pro' },
  { value: 'google/gemini-2.5-flash',      label: 'Gemini 2.5 Flash' },
  { value: 'google/gemini-2.5-flash-lite', label: 'Gemini 2.5 Flash-Lite' },
]

const EXAMPLE_PROMPTS = [
  { label: 'Why is this stuck?', text: 'Why is this study stuck and what should I do to fix it?' },
  { label: 'Routing issue?', text: 'Why did this study fail to route to its destination?' },
  { label: 'Pipeline failed?', text: 'Which pipeline steps failed and what are the errors?' },
  { label: 'Defacing OK?', text: 'Was defacing successful and what is the QA score?' },
  { label: 'Approve this?', text: 'Should I approve this study? Are there any blockers or concerns?' },
  { label: 'What happened?', text: 'Summarize everything that has happened to this study so far.' },
]

export function AgentPanel({ prefillStudyId, prefillStudyUid }: AgentPanelProps) {
  const baseUrl = import.meta.env.VITE_AGENT_BASE_URL || DEFAULT_AGENT_BASE_URL
  const [studyId, setStudyId] = useState('')
  const [studyUid, setStudyUid] = useState('')
  const [question, setQuestion] = useState('')
  const [includeNextSteps, setIncludeNextSteps] = useState(true)
  const [agentApiKey, setAgentApiKey] = useState('')
  const [model, setModel] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<AgentResult | null>(null)
  const [availableModels, setAvailableModels] = useState<{ value: string; label: string }[]>(FALLBACK_MODELS)

  useEffect(() => {
    fetch(`${baseUrl}/info`)
      .then((r) => r.ok ? r.json() : null)
      .then((json) => {
        if (json?.data?.models?.length) setAvailableModels(json.data.models)
      })
      .catch(() => { /* keep fallback */ })
  }, [baseUrl])

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
      include_next_steps: includeNextSteps,
      model: model || undefined
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
    setModel('')
    setError(null)
    setResult(null)
  }

  const data = result && result.ok ? result.data : null

  return (
    <div className="agent-panel">
      <div className="agent-panel__header">
        <div>
          <h2>AEGIS Agent</h2>
          <p>
            AI-powered study diagnostics — ask why a study is stuck, why routing failed, what pipeline steps ran, and more.
            Paste a study UUID or StudyInstanceUID, then ask a question or pick an example below.
          </p>
        </div>
      </div>

      <div className="agent-form">
        <div className="form-row">
          <input
            className="aegis-filter"
            type="text"
            placeholder="Study UUID (e.g. 7445e605-…)"
            value={studyId}
            onChange={(e) => setStudyId(e.target.value)}
            title="The database UUID for the study — copy it from the study detail page"
          />
          <input
            className="aegis-filter"
            type="text"
            placeholder="StudyInstanceUID (e.g. 1.2.826.0.1…)"
            value={studyUid}
            onChange={(e) => setStudyUid(e.target.value)}
            title="The DICOM StudyInstanceUID — copy it from the Synth Generator or study list"
          />
        </div>
        <div className="form-row">
          <input
            className="aegis-filter"
            type="text"
            placeholder="Ask a question — or pick one below ↓"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            title="Leave blank for a general status summary, or type a specific question"
          />
        </div>

        {/* Example prompt chips */}
        <div className="agent-prompts">
          <span className="agent-prompts__label">Try:</span>
          {EXAMPLE_PROMPTS.map((p) => (
            <button
              key={p.label}
              type="button"
              className={`agent-prompt-chip${question === p.text ? ' agent-prompt-chip--active' : ''}`}
              onClick={() => setQuestion(question === p.text ? '' : p.text)}
              title={p.text}
            >
              {p.label}
            </button>
          ))}
        </div>

        <div className="form-row">
          <input
            className="aegis-filter"
            type="password"
            placeholder="Agent API key (optional)"
            value={agentApiKey}
            onChange={(e) => setAgentApiKey(e.target.value)}
            title="Leave blank to use the server-configured API key, or enter your own for rate-limiting purposes"
          />
          <select
            className="aegis-filter"
            value={model}
            onChange={(e) => setModel(e.target.value)}
            title="'Server default' uses the model configured on the MCP server."
          >
            <option value="">Server default</option>
            {availableModels.map((m) => (
              <option key={m.value} value={m.value}>{m.label}</option>
            ))}
          </select>
        </div>
        <label className="form-checkbox" title="Ask the agent to suggest concrete remediation steps based on what it finds">
          <input
            type="checkbox"
            checked={includeNextSteps}
            onChange={(e) => setIncludeNextSteps(e.target.checked)}
          />
          Include next steps
        </label>
        <div className="aegis-form-actions">
          <button type="button" className="aegis-btn-primary" onClick={submit} disabled={loading}>
            {loading ? 'Running…' : 'Ask agent'}
          </button>
          <button type="button" className="aegis-btn-secondary" onClick={resetForm} disabled={loading}>
            Clear
          </button>
        </div>
        {error && <div className="aegis-error">{error}</div>}
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
