import { Outlet, Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import { useAuth } from "../hooks/useAuth";

// Layout は認証後の画面の共通シェルを提供するコンポーネント。
// 上部にヘッダー (ロゴ + ログアウトボタン)、下部にナビ (ホーム / グループ)、
// 中央の <Outlet /> に各ページが描画される。
export function Layout() {
  const { data: user } = useAuth();
  const navigate = useNavigate();
  const qc = useQueryClient();

  // ログアウト処理 (useMutation で POST /api/auth/logout を呼ぶ)
  const logout = useMutation({
    mutationFn: () => api.post<void>("/api/auth/logout"),
    onSuccess: () => {
      // キャッシュを全消去 (前ユーザーのデータを残さない)
      qc.clear();
      // /login にリダイレクト
      navigate("/login");
    },
  });

  return (
    <div className="min-h-screen bg-gray-100 pb-20">
      {/* 上部ヘッダー: ロゴ + ユーザー情報 + ログアウトボタン */}
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

      {/* メインコンテンツ: ここに各ページ (Timeline / Groups 等) が <Outlet /> で挿入される */}
      <main className="max-w-2xl mx-auto px-4 py-6">
        <Outlet />
      </main>

      {/* 下部ナビゲーション: 主要ページへのリンク */}
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
