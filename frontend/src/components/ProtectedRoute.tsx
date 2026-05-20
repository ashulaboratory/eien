import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

// 認証必須ルートをラップ。未認証なら /login にリダイレクト。
export function ProtectedRoute() {
  const { data: user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center text-gray-500">
        読み込み中…
      </div>
    );
  }
  if (!user) return <Navigate to="/login" replace />;
  return <Outlet />;
}
