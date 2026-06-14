import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { configureLocalIDClient } from "@rqc-icu/localid-client";
import App from "./App";
import { getRuntimeConfig } from "./runtime-config";
import "./App.css";

const { agentUrl, backendUrl } = getRuntimeConfig();

configureLocalIDClient({
  agentUrl,
  backendUrl,
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
