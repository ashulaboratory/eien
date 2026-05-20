import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import type { InvitePreview } from "../api/types";

export function InviteAccept() {
  const { token } = useParams<{ token: string }>();
  const inviteToken = token!;
  const { data: user, isLoading: authLoading } = useAuth();
  const [displayName, setDisplayName] = useState("");
  const navigate = useNavigate();

  const preview = useQuery({
    queryKey: ["invite", inviteToken],
    queryFn: () => api.get<InvitePreview>(`/api/invites/${encodeURIComponent(inviteToken)}`),
  });

  const accept = useMutation({
    mutationFn: () =>
      api.post(`/api/invites/${encodeURIComponent(inviteToken)}/accept`, {
        display_name: displayName,
      }),
    onSuccess: () => navigate("/"),
  });

  if (preview.isLoading || authLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center text-gray-500">
        読み込み中…
      </div>
    );
  }

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
        <div className="text-center">
          <p className="text-sm text-gray-500">招待されました</p>
          <h1 className="text-2xl font-bold mt-1">{preview.data?.group_name}</h1>
          <p className="text-xs text-gray-500 mt-1">
            有効期限: {new Date(preview.data?.expires_at ?? "").toLocaleDateString("ja-JP")}
          </p>
        </div>

        {!user ? (
          <div className="space-y-3 pt-4 border-t">
            <p className="text-sm text-gray-700 text-center">
              参加するにはログインまたは新規登録が必要です
            </p>
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
          <form
            onSubmit={(e) => {
              e.preventDefault();
              accept.mutate();
            }}
            className="space-y-3 pt-4 border-t"
          >
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
