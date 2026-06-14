export interface HealthResponse {
  ok: boolean;
  name: string;
  version: string;
}

export interface StatusResponse {
  provider: string;
  ready: boolean;
  cardPresent: boolean;
}

export interface SignChallengeRequest {
  challenge: string;
  backend: string;
  purpose: string;
  origin: string;
}

export interface SignChallengeResponse {
  provider: string;
  algorithm: string;
  challenge: string;
  signature: string;
  certificate?: string;
  signedAt: string;
}

export interface ChallengeResponse {
  challenge: string;
}

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
