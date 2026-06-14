export const AUTH_FLOW_DIAGRAM = `sequenceDiagram
  participant Frontend
  participant Backend
  participant Agent as Agent (localhost)

  Frontend->>Backend: POST /localid/challenge
  Backend-->>Frontend: { challenge }
  Frontend->>Agent: POST /sign-challenge
  Agent-->>Frontend: { signature, certificate, signedAt }
  Frontend->>Backend: POST /localid/verify
  Backend-->>Frontend: { success, user }`;

export const INTEGRATION_OVERVIEW_DIAGRAM = `flowchart LR
  subgraph browser [Browser]
    Frontend[Your frontend]
  end

  subgraph server [Your infrastructure]
    Backend[Your backend API]
  end

  subgraph local [This machine]
    Agent[LocalID Agent]
    Card[Smartcard / eID]
  end

  Frontend -->|1 challenge| Backend
  Frontend -->|2 sign| Agent
  Agent --> Card
  Frontend -->|3 verify proof| Backend`;
