// アプリのエントリーポイント。
// React アプリを #root にマウントし、TanStack Query の Provider で App をラップする。
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import "./index.css";
import App from "./App.tsx";
import { queryClient } from "./lib/queryClient";

// index.html の <div id="root"> に React アプリをマウント
createRoot(document.getElementById("root")!).render(
  // StrictMode: 開発時の二重実行で副作用や問題を検出する React の機能
  <StrictMode>
    {/* QueryClientProvider: アプリ全体に TanStack Query のキャッシュを供給する */}
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
