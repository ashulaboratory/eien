// Layout は認証後の全画面の共通シェル (Warm Album デザイン適用)。
// 上部ヘッダー (ロゴ + ログアウト) + 中央コンテンツ + 下部ナビ (5項目)。
import { Outlet, NavLink, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import { useAuth } from "../hooks/useAuth";

// アイコン用 SVG (lucide スタイルのインラインアイコン、依存追加せず軽量に)
function IconHome() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="w-5 h-5">
      <path d="M3 10.5L12 3l9 7.5" />
      <path d="M5 9.5V21h14V9.5" />
      <path d="M10 21v-6h4v6" />
    </svg>
  );
}
function IconChat() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="w-5 h-5">
      <path d="M21 12a8 8 0 0 1-11.5 7.2L4 21l1.8-5.5A8 8 0 1 1 21 12z" />
    </svg>
  );
}
function IconPlus() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="w-5 h-5">
      <path d="M12 5v14M5 12h14" />
    </svg>
  );
}
function IconGroups() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="w-5 h-5">
      <circle cx="9" cy="9" r="3" />
      <path d="M3 20c0-3 2.5-5 6-5s6 2 6 5" />
      <circle cx="17" cy="8" r="2.5" />
      <path d="M16 14c2 0 5 1.5 5 4.5" />
    </svg>
  );
}
function IconUser() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className="w-5 h-5">
      <circle cx="12" cy="8" r="3.5" />
      <path d="M5 20c0-3.5 3-6 7-6s7 2.5 7 6" />
    </svg>
  );
}

type NavItem = { to: string; label: string; icon: React.ReactNode };
const NAV_ITEMS: NavItem[] = [
  { to: "/", label: "ホーム", icon: <IconHome /> },
  { to: "/chat", label: "チャット", icon: <IconChat /> },
  { to: "/posts/new", label: "投稿", icon: <IconPlus /> },
  { to: "/groups", label: "グループ", icon: <IconGroups /> },
  { to: "/account", label: "アカウント", icon: <IconUser /> },
];

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
    // pb-24: 下部ナビの高さ分の余白を本文に確保
    <div className="min-h-screen bg-cream pb-24">
      {/* ヘッダー: クリーム背景 + 暖色罫線 (sticky で上部固定) */}
      <header className="bg-cream-soft border-b border-border-warm sticky top-0 z-10">
        <div className="max-w-2xl mx-auto px-4 py-3 flex justify-between items-center">
          {/* ロゴ画像: /public/logo.png を参照 */}
          <NavLink to="/" className="flex items-center" aria-label="Eien">
            <img
              src="/logo.png"
              alt="Eien"
              className="h-9 w-auto"
            />
          </NavLink>

          {user && (
            <div className="flex items-center gap-3 text-sm">
              <span className="text-ink-muted hidden sm:inline">
                {user.email}
              </span>
              <button
                onClick={() => logout.mutate()}
                className="text-ink-muted hover:text-terracotta transition-colors"
              >
                ログアウト
              </button>
            </div>
          )}
        </div>
      </header>

      {/* 本文 */}
      <main className="max-w-2xl mx-auto px-4 py-6">
        <Outlet />
      </main>

      {/* ボトムナビ: 5 項目 (ホーム / チャット / 投稿 / グループ / アカウント) */}
      <nav className="fixed bottom-0 left-0 right-0 bg-cream-soft border-t border-border-warm">
        <div className="max-w-2xl mx-auto px-2 py-2 flex justify-around">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              // NavLink は active 状態を className 関数で受け取れる
              className={({ isActive }) =>
                [
                  "flex flex-col items-center gap-1 px-3 py-1 text-[11px] font-medium transition-colors",
                  isActive ? "text-terracotta" : "text-ink-muted hover:text-terracotta",
                ].join(" ")
              }
            >
              {item.icon}
              <span>{item.label}</span>
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  );
}
