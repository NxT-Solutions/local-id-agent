import { useEffect } from "react";
import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAdminLock } from "@/context/AdminLockContext";

export function AdminRoute() {
  const { unlocked, loading, requestUnlock } = useAdminLock();
  const location = useLocation();

  useEffect(() => {
    if (!loading && !unlocked) {
      requestUnlock();
    }
  }, [loading, unlocked, requestUnlock]);

  if (loading) {
    return null;
  }

  if (!unlocked) {
    return (
      <Navigate
        to="/"
        replace
        state={{ adminRequired: true, from: location.pathname }}
      />
    );
  }

  return <Outlet />;
}
