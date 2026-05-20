import { Outlet, Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import { useAuth } from "../hooks/useAuth";

// 認証後の画面の共通シェル: 上ヘッダー + 下ナビ
export function Layout() {
  const { data: user } = useAuth();
  const navigate = useNavigate();
  const qc = useQueryClient();

  const logout = useMutation({
    mutationFn: () => api.post<void>("/api/auth/logout"),
    onSuccess: () => {
      qc.clear();
      navigate("/login");
    },
  });

  return (
    <div className="min-h-screen bg-gray-100 pb-20">
      <header className="bg-white border-b sticky top-0 z-10">
        <div className="max-w-2xl mx-auto px-4 py-3 flex justify-between items-center">
          <Link to="/" className="text-2xl font-bold text-blue-600">
            Eien
          </Link>
          {user && (
            <div className="flex items-center gap-3 text-sm">
              <span className="text-gray-600 hidden sm:inline">{user.email}</span>
              <button
                onClick={() => logout.mutate()}
                className="text-gray-500 hover:text-gray-700"
              >
                ログアウト
              </button>
            </div>
          )}
        </div>
      </header>

      <main className="max-w-2xl mx-auto px-4 py-6">
        <Outlet />
      </main>

      <nav className="fixed bottom-0 left-0 right-0 bg-white border-t">
        <div className="max-w-2xl mx-auto px-4 py-2 flex justify-around">
          <Link to="/" className="text-gray-700 hover:text-blue-600 text-sm py-2">
            🏠 ホーム
          </Link>
          <Link to="/groups" className="text-gray-700 hover:text-blue-600 text-sm py-2">
            👥 グループ
          </Link>
        </div>
      </nav>
    </div>
  );
}
