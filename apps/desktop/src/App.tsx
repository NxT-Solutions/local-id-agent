import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { configureLocalIDClient } from "@rqc-icu/localid-client";
import { AppLayout } from "@/components/AppLayout";
import { AboutPage } from "@/pages/AboutPage";
import { DashboardPage } from "@/pages/DashboardPage";
import { DemoPage } from "@/pages/DemoPage";
import { SetupPage } from "@/pages/SetupPage";
import { SettingsPage } from "@/pages/SettingsPage";

configureLocalIDClient({
  agentUrl: import.meta.env.VITE_AGENT_URL ?? "http://127.0.0.1:17443",
  backendUrl: import.meta.env.VITE_BACKEND_URL ?? "http://localhost:8000",
});

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<DashboardPage />} />
          <Route path="setup" element={<SetupPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="about" element={<AboutPage />} />
          <Route path="demo" element={<DemoPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
