export {
  configureLocalIDClient,
  getAgentUrl,
  getBackendUrl,
} from "./config";
export type { LocalIDClientConfig } from "./config";

export {
  checkAgentReadiness,
  fetchHealth,
  fetchStatus,
  signChallenge,
} from "./agent";

export { fetchChallenge, verifyProof } from "./backend";

export type {
  AgentReadiness,
  AuthState,
  ChallengeResponse,
  HealthResponse,
  SignChallengeRequest,
  SignChallengeResponse,
  StatusResponse,
  VerifyRequest,
  VerifyResponse,
} from "./types";

export * as agentOpenAPI from "./openapi/agent";
export * as backendOpenAPI from "./openapi/backend";
export type {
  HealthResponse as OpenAPIHealthResponse,
  Status as OpenAPIStatus,
  SignChallengeRequest as OpenAPISignChallengeRequest,
  SignChallengeResponse as OpenAPISignChallengeResponse,
} from "./openapi/agent";
export type {
  ChallengeResponse as OpenAPIChallengeResponse,
  VerifyRequest as OpenAPIVerifyRequest,
  VerifyResponse as OpenAPIVerifyResponse,
} from "./openapi/backend";
