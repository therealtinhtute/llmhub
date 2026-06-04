import { useEffect, useRef, useState, type ReactElement } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/stores';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';

export function ProtectedRoute({ children }: { children: ReactElement }) {
  const location = useLocation();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const managementKey = useAuthStore((state) => state.managementKey);
  const apiBase = useAuthStore((state) => state.apiBase);
  const checkAuth = useAuthStore((state) => state.checkAuth);
  const [checking, setChecking] = useState(false);
  const lastAttemptRef = useRef<string>('');

  useEffect(() => {
    const attemptKey = `${apiBase}::${managementKey}`;
    if (isAuthenticated || !managementKey || !apiBase) {
      lastAttemptRef.current = '';
      setChecking(false);
      return;
    }

    if (lastAttemptRef.current === attemptKey) {
      return;
    }

    lastAttemptRef.current = attemptKey;
    let cancelled = false;

    const tryRestore = async () => {
      setChecking(true);
      try {
        await checkAuth();
      } finally {
        if (!cancelled) {
          setChecking(false);
        }
      }
    };
    tryRestore();

    return () => {
      cancelled = true;
    };
  }, [apiBase, isAuthenticated, managementKey, checkAuth]);

  if (checking) {
    return (
      <div className="main-content">
        <LoadingSpinner />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }

  return children;
}
