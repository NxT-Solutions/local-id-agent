import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { configureLocalIDClient } from "@rqc-icu/localid-client";
import { AdminRoute } from "@/components/admin/AdminRoute";
import { AppLayout } from "@/components/AppLayout";
import { AdminLockProvider, useAdminLock } from "@/context/AdminLockContext";
import { AboutPage } from "@/pages/AboutPage";
import { AdminSetupPage } from "@/pages/AdminSetupPage";
import { DashboardPage } from "@/pages/DashboardPage";
import { DemoPage } from "@/pages/DemoPage";
import { SetupPage } from "@/pages/SetupPage";
import { SettingsPage } from "@/pages/SettingsPage";
import { appFetch } from "@/lib/fetch";

configureLocalIDClient({
  agentUrl: import.meta.env.VITE_AGENT_URL ?? "http://127.0.0.1:17443",
  backendUrl: import.meta.env.VITE_BACKEND_URL ?? "http://localhost:8000",
  fetchImpl: appFetch,
});

function AppRoutes() {
  const { setupRequired, loading } = useAdminLock();

  if (loading) {
    return null;
  }

  if (setupRequired) {
    return <AdminSetupPage />;
  }

  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="about" element={<AboutPage />} />
        <Route element={<AdminRoute />}>
          <Route path="setup" element={<SetupPage />} />
          <Route path="demo" element={<DemoPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AdminLockProvider>
        <AppRoutes />
      </AdminLockProvider>
    </BrowserRouter>
  );
}
