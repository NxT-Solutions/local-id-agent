import type {
  HealthResponse as GenHealthResponse,
  SignChallengeRequest as GenSignChallengeRequest,
  SignChallengeResponse as GenSignChallengeResponse,
  Status as GenStatus,
} from "./generated/localid/v1/protocol_pb";

/** Plain JSON wire shape for protobuf messages (strips runtime Message metadata). */
type JsonProto<T> = Omit<T, "$typeName" | "$unknown">;

export type HealthResponse = JsonProto<GenHealthResponse>;
export type SignChallengeRequest = JsonProto<GenSignChallengeRequest>;
export type SignChallengeResponse = JsonProto<GenSignChallengeResponse>;
export type StatusResponse = JsonProto<GenStatus>;

/** Backend challenge payload (mock/Laravel contract; not part of the agent API). */
export interface ChallengeResponse {
  challenge: string;
}

/** Proof verification payload sent to the backend. */
export interface VerifyRequest {
  challenge: string;
  backend: string;
  origin: string;
  purpose: string;
  provider: string;
  algorithm: string;
  signature: string;
  certificate: string;
  signedAt: string;
}

export interface VerifyResponse {
  success: boolean;
  user: {
    id: string;
    name: string;
  };
}

export interface AgentReadiness {
  healthy: boolean;
  ready: boolean;
  provider: string;
  error?: string;
}

export type AuthState = "idle" | "loading" | "success" | "error";
