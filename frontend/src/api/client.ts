// API 呼び出しの共通ラッパー。
// - Cookie の自動送信 (credentials: "include")
// - JSON / FormData の Content-Type 自動設定
// - エラーレスポンスを ApiError として throw

// ApiError は HTTP ステータスを保持する独自エラークラス。
// 401 などのステータスを呼び出し側で判定するために使う (例: useAuth)。
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

// request は内部用の汎用 HTTP 呼び出し関数。
// ジェネリクス <T> でレスポンスの型を呼び出し側が指定できる。
//   例: api.get<User>("/api/auth/me") → User 型として扱われる
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  // ヘッダの組み立て: 呼び出し元の指定を引き継ぎつつ、必要なら Content-Type を補う
  const headers: Record<string, string> = { ...(init?.headers as Record<string, string>) };
  // body が FormData じゃない場合 (= JSON 想定) は、Content-Type を JSON に設定
  // FormData の時は fetch が自動でマルチパート用の Content-Type を付けるため明示しない
  if (init?.body && !(init.body instanceof FormData) && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }

  // fetch でリクエスト送信
  // credentials: "include" は Cookie (session_id) を必ず送るための重要設定
  const res = await fetch(path, {
    ...init,
    credentials: "include",
    headers,
  });

  // エラーレスポンス (4xx / 5xx) なら ApiError を throw
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      // バックエンドが返す JSON の "error" フィールドからメッセージ取得
      const data = await res.json();
      if (data?.error) msg = typeof data.error === "string" ? data.error : msg;
    } catch {
      // JSON でない場合はステータスコードだけメッセージに使う
    }
    throw new ApiError(res.status, msg);
  }

  // 204 No Content (ログアウトなど) はボディがないので undefined を返す
  if (res.status === 204) return undefined as T;
  // それ以外は JSON をパースして返す
  return res.json();
}

// api は HTTP メソッド別の簡易呼び出し関数をまとめたオブジェクト。
// 呼び出し側は api.get / api.post / api.delete を使う。
export const api = {
  // GET: シンプルにパスだけ渡す
  get: <T>(path: string) => request<T>(path),
  // POST: body が FormData ならそのまま、それ以外は JSON 化して送る
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: "POST",
      body: body instanceof FormData ? body : body ? JSON.stringify(body) : undefined,
    }),
  // DELETE: body なしで送る
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
