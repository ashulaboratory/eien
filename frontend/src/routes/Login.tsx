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
      qc.invalidateQueries({ queryKey: ["auth"] });
      navigate("/");
    },
  });

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100 p-4">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          login.mutate();
        }}
        className="bg-white rounded-lg shadow-lg p-8 w-full max-w-md space-y-4"
      >
        <h1 className="text-3xl font-bold text-blue-600">Eien</h1>
        <p className="text-sm text-gray-600 mb-2">永遠の仲間とのSNS</p>

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
          <label className="block text-sm text-gray-700 mb-1">パスワード</label>
          <input
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        {login.error && (
          <p className="text-red-600 text-sm">{(login.error as Error).message}</p>
        )}

        <button
          type="submit"
          disabled={login.isPending}
          className="w-full bg-blue-600 text-white py-2 rounded font-medium hover:bg-blue-700 disabled:opacity-50"
        >
          {login.isPending ? "ログイン中…" : "ログイン"}
        </button>

        <p className="text-sm text-gray-600 text-center pt-2">
          アカウントがない方は{" "}
          <Link to="/register" className="text-blue-600 hover:underline">
            こちら
          </Link>
        </p>
      </form>
    </div>
  );
}
