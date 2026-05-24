// Login は Eien のログイン画面 (Warm Album デザイン適用)。
// メール+パスワードを /api/auth/login にPOSTし、成功すれば Cookie が発行されてホームへ。
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { User } from "../api/types";

export function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const navigate = useNavigate();
  const qc = useQueryClient();

  const login = useMutation({
    mutationFn: () => api.post<User>("/api/auth/login", { email, password }),
    onSuccess: () => {
      // ログイン成功 → useAuth キャッシュを無効化して再fetch → ホームへ遷移
      qc.invalidateQueries({ queryKey: ["auth"] });
      navigate("/");
    },
  });

  return (
    <div className="min-h-screen flex items-center justify-center bg-cream p-4">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          login.mutate();
        }}
        className="
          bg-cream-soft border border-border-warm rounded-2xl
          p-10 w-full max-w-md space-y-5
        "
      >
        {/* タイトル: セリフで重厚さを出す。トラッキングを少し詰めて密度感 */}
        <div className="text-center">
          <h1 className="font-serif text-4xl text-ink tracking-tight">Eien</h1>
          <p className="text-sm text-ink-muted mt-1">
            永遠の仲間とのSNS
          </p>
        </div>

        <div className="border-t border-border-warm pt-5 space-y-4">
          <div>
            <label className="block text-sm text-ink-muted mb-1">
              メールアドレス
            </label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="
                w-full px-3 py-2 bg-cream border border-border-warm rounded-lg
                text-ink placeholder:text-ink-muted/60
                focus:outline-none focus:border-terracotta
                transition-colors
              "
            />
          </div>

          <div>
            <label className="block text-sm text-ink-muted mb-1">
              パスワード
            </label>
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="
                w-full px-3 py-2 bg-cream border border-border-warm rounded-lg
                text-ink
                focus:outline-none focus:border-terracotta
                transition-colors
              "
            />
          </div>

          {login.error && (
            <p className="text-sm text-rose">
              {(login.error as Error).message}
            </p>
          )}

          <button
            type="submit"
            disabled={login.isPending}
            className="
              w-full bg-terracotta text-cream-soft py-2.5 rounded-full
              font-medium tracking-wide
              hover:bg-terracotta-dark
              disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors
            "
          >
            {login.isPending ? "ログイン中…" : "ログイン"}
          </button>
        </div>

        <p className="text-sm text-ink-muted text-center">
          アカウントがない方は{" "}
          <Link
            to="/register"
            className="text-terracotta hover:text-terracotta-dark underline underline-offset-2"
          >
            こちら
          </Link>
        </p>
      </form>
    </div>
  );
}
