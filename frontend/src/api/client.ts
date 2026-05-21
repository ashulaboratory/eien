// API呼び出しのラッパー。Cookie 自動送信 + エラーハンドリング。

// Errorを拡張してstatusも持てるようにしたクラス
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

//API呼び出しのラッパー関数
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { ...(init?.headers as Record<string, string>) };
  if (init?.body && !(init.body instanceof FormData) && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }

  const res = await fetch(path, {
    ...init,
    credentials: "include", // Cookie送信に必須
    headers,
  });

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const data = await res.json();
      if (data?.error) msg = typeof data.error === "string" ? data.error : msg;
    } catch {
      // JSON でない場合はステータスのみ
    }
    throw new ApiError(res.status, msg);
  }
  
  //204の場合はundefinedを返す。
  if (res.status === 204) return undefined as T;
  return res.json();
}

//API呼び出しのラッパー関数をオブジェクトにまとめたもの
export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "POST",
      body: body instanceof FormData ? body : body ? JSON.stringify(body) : undefined,
    }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
