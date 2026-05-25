import { AgentPanel } from '../components/AgentPanel'

// AgentPage renders the AI Agent on its own top-level researcher route at
// /agent. Same panel that lives at /admin/agent — we don't gate it
// behind /admin/* because the agent answers general questions about
// projects and studies that researchers (not just operators) routinely
// need, and the user explicitly asked for "agent available from the
// home screen."
export function AgentPage() {
  return (
    <>
      <header className="aegis-page-header">
        <div className="aegis-page-header-row">
          <div>
            <h1>AI Agent</h1>
            <p className="aegis-page-description">
              Ask questions about studies, pipeline state, and routing in natural language.
              The agent has read access to the live AEGIS API.
            </p>
          </div>
        </div>
      </header>
      <AgentPanel />
    </>
  )
}
