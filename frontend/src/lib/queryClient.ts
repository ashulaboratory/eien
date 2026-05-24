import { QueryClient } from "@tanstack/react-query";

// TanStack Query のクライアントインスタンス。
// main.tsx の QueryClientProvider で App 全体に供給される。
// すべての useQuery / useMutation はこのインスタンスのキャッシュを共有する。
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // retry: false → 401 などのエラーで自動再試行しない
      // (認証エラーを何度もリトライしても意味ないため)
      retry: false,
      // staleTime: 30秒 → この間は同じ queryKey の再フェッチをしない
      // (短時間の連続アクセスで API を叩きまくらないため)
      staleTime: 30 * 1000,
      // refetchOnWindowFocus: false → タブにフォーカスが戻った時の自動再取得を無効化
      // (デフォルトは true だが、Eien では明示的な操作で取得する設計)
      refetchOnWindowFocus: false,
    },
  },
});
