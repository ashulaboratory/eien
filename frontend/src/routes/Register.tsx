// Register は Eien の新規登録画面 (Warm Album デザイン適用)。
// メール+パスワード+誕生日 を /api/auth/register にPOSTし、
// 成功時はバックエンドが自動でセッションを発行 → ホームへ遷移。
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { User } from "../api/types";

export function Register() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [birthday, setBirthday] = useState("");
  const navigate = useNavigate();
  const qc = useQueryClient();

  const register = useMutation({
    mutationFn: () =>
      api.post<User>("/api/auth/register", { email, password, birthday }),
    onSuccess: () => {
      // 登録成功 → useAuth キャッシュを無効化して再fetch → ホームへ遷移
      qc.invalidateQueries({ queryKey: ["auth"] });
      navigate("/");
    },
  });

  return (
    <div className="min-h-screen flex items-center justify-center bg-cream p-4">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          register.mutate();
        }}
        className="
          bg-cream-soft border border-border-warm rounded-2xl
          p-10 w-full max-w-md space-y-5
        "
      >
        {/* タイトル: Login と統一感を出すため同じ書式 */}
        <div className="text-center">
          <h1 className="font-serif text-4xl text-ink tracking-tight">Eien</h1>
          <p className="text-sm text-ink-muted mt-1">新規登録</p>
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
              <span className="text-xs text-ink-muted/70 ml-1">(8文字以上)</span>
            </label>
            <input
              type="password"
              required
              minLength={8}
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

          <div>
            <label className="block text-sm text-ink-muted mb-1">
              誕生日
            </label>
            <input
              type="date"
              required
              value={birthday}
              onChange={(e) => setBirthday(e.target.value)}
              className="
                w-full px-3 py-2 bg-cream border border-border-warm rounded-lg
                text-ink
                focus:outline-none focus:border-terracotta
                transition-colors
              "
            />
            <p className="text-xs text-ink-muted/80 mt-1">
              誕生日通知に使われます
            </p>
          </div>

          {register.error && (
            <p className="text-sm text-rose">
              {(register.error as Error).message}
            </p>
          )}

          <button
            type="submit"
            disabled={register.isPending}
            className="
              w-full bg-terracotta text-cream-soft py-2.5 rounded-full
              font-medium tracking-wide
              hover:bg-terracotta-dark
              disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors
            "
          >
            {register.isPending ? "登録中…" : "登録してログイン"}
          </button>
        </div>

        <p className="text-sm text-ink-muted text-center">
          すでにアカウントをお持ちの方は{" "}
          <Link
            to="/login"
            className="text-terracotta hover:text-terracotta-dark underline underline-offset-2"
          >
            ログイン
          </Link>
        </p>
      </form>
    </div>
  );
}
