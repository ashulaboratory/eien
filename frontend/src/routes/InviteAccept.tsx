import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import type { InvitePreview } from "../api/types";

// InviteAccept は招待リンクからグループに参加する画面のコンポーネント。
// /invite/:token でアクセスされ、認証不要 (バックエンドが認証不要に設定)。
//
// 処理の流れ:
// 1. URL の token から招待プレビューを取得 (グループ名・有効期限など)
// 2. 未ログインなら「ログイン or 新規登録」を促す
// 3. ログイン済みなら表示名入力欄を出し、参加処理を実行
export function InviteAccept() {
  // URL パラメータから招待トークンを取得
  const { token } = useParams<{ token: string }>();
  const inviteToken = token!;
  // 認証状態を取得 (ログインしているかどうかで UI を切り替える)
  const { data: user, isLoading: authLoading } = useAuth();
  const [displayName, setDisplayName] = useState("");
  const navigate = useNavigate();

  // 招待リンクのプレビュー情報を取得 (認証不要のエンドポイント)
  const preview = useQuery({
    queryKey: ["invite", inviteToken],
    queryFn: () => api.get<InvitePreview>(`/api/invites/${encodeURIComponent(inviteToken)}`),
  });

  // 招待を承諾する処理 (グループに参加)
  const accept = useMutation({
    mutationFn: () =>
      api.post(`/api/invites/${encodeURIComponent(inviteToken)}/accept`, {
        display_name: displayName,
      }),
    onSuccess: () => navigate("/"), // 参加成功後はホームへ
  });

  // ローディング中
  if (preview.isLoading || authLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center text-gray-500">
        読み込み中…
      </div>
    );
  }

  // 招待リンクが無効 (期限切れ or 使用上限到達 or 存在しない)
  if (preview.error) {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow p-6 max-w-md text-center space-y-3">
          <p className="text-red-600 font-medium">
            この招待リンクは無効か期限切れです
          </p>
          <p className="text-sm text-gray-500">
            {(preview.error as Error).message}
          </p>
          <Link to="/" className="text-blue-600 text-sm hover:underline">
            ホームへ戻る
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100 p-4">
      <div className="bg-white rounded-lg shadow-lg p-8 w-full max-w-md space-y-4">
        {/* 招待プレビュー (グループ名・有効期限) */}
        <div className="text-center">
          <p className="text-sm text-gray-500">招待されました</p>
          <h1 className="text-2xl font-bold mt-1">{preview.data?.group_name}</h1>
          <p className="text-xs text-gray-500 mt-1">
            有効期限: {new Date(preview.data?.expires_at ?? "").toLocaleDateString("ja-JP")}
          </p>
        </div>

        {/* 未ログイン時: ログイン or 新規登録への誘導 */}
        {!user ? (
          <div className="space-y-3 pt-4 border-t">
            <p className="text-sm text-gray-700 text-center">
              参加するにはログインまたは新規登録が必要です
            </p>
            {/* state で元URLを渡し、ログイン後に戻れるようにする */}
            <Link
              to="/login"
              state={{ from: `/invite/${inviteToken}` }}
              className="block bg-blue-600 text-white text-center py-2 rounded hover:bg-blue-700"
            >
              ログイン
            </Link>
            <Link
              to="/register"
              state={{ from: `/invite/${inviteToken}` }}
              className="block bg-white border border-blue-600 text-blue-600 text-center py-2 rounded hover:bg-blue-50"
            >
              新規登録
            </Link>
          </div>
        ) : (
          // ログイン済み: 表示名入力 + 参加ボタン
          <form
            onSubmit={(e) => {
              e.preventDefault();
              accept.mutate();
            }}
            className="space-y-3 pt-4 border-t"
          >
            {/* グループでの表示名 (Eien 固有: グループごとに名乗りを変えられる) */}
            <div>
              <label className="block text-sm text-gray-700 mb-1">
                このグループでの表示名
              </label>
              <input
                type="text"
                required
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="例: たかし"
                className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            {accept.error && (
              <p className="text-red-600 text-sm">{(accept.error as Error).message}</p>
            )}

            <button
              type="submit"
              disabled={accept.isPending}
              className="w-full bg-blue-600 text-white py-2 rounded font-medium hover:bg-blue-700 disabled:opacity-50"
            >
              {accept.isPending ? "参加中…" : "参加する"}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
