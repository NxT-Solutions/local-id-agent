<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  checkAgentReadiness,
  getBackendUrl,
  signChallenge,
  fetchChallenge,
  verifyProof,
} from "@rqc-icu/localid-client";
import type { AgentReadiness, AuthState } from "@rqc-icu/localid-client";

const agent = ref<AgentReadiness | null>(null);
const authState = ref<AuthState>("idle");
const userName = ref<string | null>(null);
const error = ref<string | null>(null);

const agentReady = computed(() => Boolean(agent.value?.healthy && agent.value?.ready));

async function refreshAgent() {
  agent.value = await checkAgentReadiness();
}

onMounted(() => {
  void refreshAgent();
});

async function handleAuthenticate() {
  authState.value = "loading";
  error.value = null;
  userName.value = null;

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

    userName.value = result.user.name;
    authState.value = "success";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Authentication failed";
    authState.value = "error";
  }
}
</script>

<template>
  <main class="app">
    <header>
      <h1>LocalID Vue Example</h1>
      <p class="subtitle">
        Browser demo for challenge signing via the LocalID Agent.
      </p>
    </header>

    <section class="panel">
      <h2>Agent status</h2>
      <p v-if="agent === null">Checking agent…</p>
      <p v-else-if="agent.error" class="status error">
        Agent unreachable: {{ agent.error }}
      </p>
      <ul v-else class="status-list">
        <li :class="agent.healthy ? 'ok' : 'bad'">
          Health: {{ agent.healthy ? "OK" : "Unavailable" }}
        </li>
        <li :class="agent.ready ? 'ok' : 'bad'">
          Provider: {{ agent.provider }} ({{ agent.ready ? "ready" : "not ready" }})
        </li>
      </ul>
      <button type="button" class="secondary" @click="refreshAgent">
        Refresh status
      </button>
    </section>

    <section class="panel">
      <h2>Authenticate</h2>
      <button
        type="button"
        class="primary"
        :disabled="!agentReady || authState === 'loading'"
        @click="handleAuthenticate"
      >
        {{ authState === "loading" ? "Authenticating…" : "Authenticate with LocalID" }}
      </button>

      <p v-if="authState === 'success' && userName" class="result success">
        Signed in as {{ userName }}
      </p>

      <p v-if="authState === 'error' && error" class="result error">
        {{ error }}
      </p>
    </section>
  </main>
</template>
