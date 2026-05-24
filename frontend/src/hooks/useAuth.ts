import { useQuery } from "@tanstack/react-query";
import { api, ApiError } from "../api/client";
import type { User } from "../api/types";

// useAuth は現在ログイン中のユーザー情報を取得するカスタムフック。
// TanStack Query 経由で /api/auth/me を呼び、結果をキャッシュ管理する。
//
// 動作:
// - 認証成功 → User オブジェクトを返す
// - 401 (未認証) → null を返す (エラーじゃなく「ログインしてない」状態として扱う)
// - その他のエラー → 例外を throw
//
// 使い方:
//   const { data: user, isLoading } = useAuth()
//   if (isLoading) ...
//   if (!user) ...
//
// queryKey ["auth", "me"] でキャッシュされ、ログイン/ログアウト時に invalidate される。
export function useAuth() {
  return useQuery<User | null>({
    queryKey: ["auth", "me"],
    queryFn: async () => {
      try {
        return await api.get<User>("/api/auth/me");
      } catch (err) {
        // 401 (未認証) は「ログインしてない」状態として null を返す
        if (err instanceof ApiError && err.status === 401) return null;
        // その他のエラーはそのまま投げる
        throw err;
      }
    },
  });
}
