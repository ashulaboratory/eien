import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false, // 401などは即座にエラー扱い (再試行しない)
      staleTime: 30 * 1000, // 30秒は再fetchしない
      refetchOnWindowFocus: false,
    },
  },
});
