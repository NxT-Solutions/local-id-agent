import { createApp } from "vue";
import { configureLocalIDClient } from "@rqc-icu/localid-client";
import App from "./App.vue";
import "./style.css";

configureLocalIDClient({
  agentUrl: import.meta.env.VITE_AGENT_URL ?? "http://127.0.0.1:17443",
  backendUrl: import.meta.env.VITE_BACKEND_URL ?? "http://localhost:8000",
});

createApp(App).mount("#app");
