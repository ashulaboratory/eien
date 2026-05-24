import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

// ProtectedRoute は認証必須ルートをラップする親コンポーネント。
// App.tsx で `<Route element={<ProtectedRoute />}>...</Route>` のように使うと、
// 内側の子ルート全てに認証チェックが適用される。
//
// 動きの3パターン:
// 1. 認証状態の読み込み中 → "読み込み中…" を表示
// 2. 未認証 (user が null) → /login にリダイレクト
// 3. 認証済み → <Outlet /> で子ルートを描画
export function ProtectedRoute() {
  // useAuth で /api/auth/me の結果を取得 (TanStack Query 経由でキャッシュ管理)
  const { data: user, isLoading } = useAuth();

  // 認証状態の読み込み中: ローディング表示
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center text-gray-500">
        読み込み中…
      </div>
    );
  }
  // 未認証: /login にリダイレクト (replace で履歴に残さない)
  if (!user) return <Navigate to="/login" replace />;
  // 認証済み: 子ルートを描画 (Outlet が子ルートの場所)
  return <Outlet />;
}
