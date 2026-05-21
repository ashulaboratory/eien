import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { User } from "../api/types";

export function Register() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [birthday, setBirthday] = useState("");

  //画面遷移するための関数
  const navigate = useNavigate();
  //アプリ全体のキャッシュ管理者を取得
  const qc = useQueryClient();

  //登録処理用のオブジェクト
  const register = useMutation({
    //実行関数
    mutationFn: () =>
      api.post<User>("/api/auth/register", { email, password, birthday }),
    //成功時の処理
    onSuccess: () => {
      //auth関連のキャッシュは古い可能性があるから取り直す。
      qc.invalidateQueries({ queryKey: ["auth"] });
      navigate("/");
    },
  });

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100 p-4">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          register.mutate();
        }}
        className="bg-white rounded-lg shadow-lg p-8 w-full max-w-md space-y-4"
      >
        <h1 className="text-3xl font-bold text-blue-600">Eien</h1>
        <p className="text-sm text-gray-600 mb-2">新規登録</p>

        <div>
          <label className="block text-sm text-gray-700 mb-1">メールアドレス</label>
          <input
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-700 mb-1">パスワード（8文字以上）</label>
          <input
            type="password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-700 mb-1">誕生日</label>
          <input
            type="date"
            required
            value={birthday}
            onChange={(e) => setBirthday(e.target.value)}
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p className="text-xs text-gray-500 mt-1">誕生日通知に使われます</p>
        </div>

        {register.error && (
          <p className="text-red-600 text-sm">{(register.error as Error).message}</p>
        )}

        <button
          type="submit"
          disabled={register.isPending}
          className="w-full bg-blue-600 text-white py-2 rounded font-medium hover:bg-blue-700 disabled:opacity-50"
        >
          {register.isPending ? "登録中…" : "登録してログイン"}
        </button>

        <p className="text-sm text-gray-600 text-center pt-2">
          すでにアカウントをお持ちの方は{" "}
          <Link to="/login" className="text-blue-600 hover:underline">
            ログイン
          </Link>
        </p>
      </form>
    </div>
  );
}
