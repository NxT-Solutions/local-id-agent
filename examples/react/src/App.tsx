import { useCallback, useEffect, useState } from "react";
import {
  checkAgentReadiness,
  getBackendUrl,
  signChallenge,
  fetchChallenge,
  verifyProof,
} from "@rqc-icu/localid-client";
import type { AgentReadiness, AuthState } from "@rqc-icu/localid-client";

function App() {
  const [agent, setAgent] = useState<AgentReadiness | null>(null);
  const [authState, setAuthState] = useState<AuthState>("idle");
  const [userName, setUserName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refreshAgent = useCallback(async () => {
    const readiness = await checkAgentReadiness();
    setAgent(readiness);
  }, []);

  useEffect(() => {
    void refreshAgent();
  }, [refreshAgent]);

  const agentReady = Boolean(agent?.healthy && agent?.ready);

  async function handleAuthenticate() {
    setAuthState("loading");
    setError(null);
    setUserName(null);

    try {
      const origin = window.location.origin;
      const backend = getBackendUrl();

      const { challenge } = await fetchChallenge(backend);
      const proof = await signChallenge({
        challenge,
        backend,
        purpose: "login",
        origin,
      });

      const result = await verifyProof(backend, {
        challenge: proof.challenge,
        backend,
        origin,
        purpose: "login",
        provider: proof.provider,
        algorithm: proof.algorithm,
        signature: proof.signature,
        certificate: proof.certificate ?? "",
        signedAt: proof.signedAt,
      });

      setUserName(result.user.name);
      setAuthState("success");
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Authentication failed";
      setError(message);
      setAuthState("error");
    }
  }

  return (
    <main className="app">
      <header>
        <h1>LocalID React Example</h1>
        <p className="subtitle">
          Browser demo for challenge signing via the LocalID Agent.
        </p>
      </header>

      <section className="panel">
        <h2>Agent status</h2>
        {agent === null ? (
          <p>Checking agent…</p>
        ) : agent.error ? (
          <p className="status error">Agent unreachable: {agent.error}</p>
        ) : (
          <ul className="status-list">
            <li className={agent.healthy ? "ok" : "bad"}>
              Health: {agent.healthy ? "OK" : "Unavailable"}
            </li>
            <li className={agent.ready ? "ok" : "bad"}>
              Provider: {agent.provider} ({agent.ready ? "ready" : "not ready"})
            </li>
          </ul>
        )}
        <button type="button" className="secondary" onClick={() => void refreshAgent()}>
          Refresh status
        </button>
      </section>

      <section className="panel">
        <h2>Authenticate</h2>
        <button
          type="button"
          className="primary"
          disabled={!agentReady || authState === "loading"}
          onClick={() => void handleAuthenticate()}
        >
          {authState === "loading" ? "Authenticating…" : "Authenticate with LocalID"}
        </button>

        {authState === "success" && userName && (
          <p className="result success">Signed in as {userName}</p>
        )}

        {authState === "error" && error && (
          <p className="result error">{error}</p>
        )}
      </section>
    </main>
  );
}

export default App;
