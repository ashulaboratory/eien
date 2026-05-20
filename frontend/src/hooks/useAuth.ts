import { useQuery } from "@tanstack/react-query";
import { api, ApiError } from "../api/client";
import type { User } from "../api/types";

// 現在ログイン中のユーザー情報を取得する hook
// 未認証なら null を返す
export function useAuth() {
  return useQuery<User | null>({
    queryKey: ["auth", "me"],
    queryFn: async () => {
      try {
        return await api.get<User>("/api/auth/me");
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) return null;
        throw err;
      }
    },
  });
}
