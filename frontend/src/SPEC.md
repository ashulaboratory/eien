# フロントエンド (frontend/src) パッケージ概要

## 責務
Eien のフロントエンドUI。バックエンドのREST APIを叩いて、Eien の機能を画面化する。

## フォルダ構成

```
src/
├── api/           # API クライアントと型定義
│   ├── client.ts     fetch ラッパー (Cookie/エラーハンドリング)
│   └── types.ts      User, Group, Post 等の TypeScript型
├── components/    # 共通UIコンポーネント
│   ├── Layout.tsx        認証後のシェル (ヘッダ + 下ナビ)
│   └── ProtectedRoute.tsx 認証ガード (未認証なら /login へ)
├── hooks/         # カスタムフック
│   └── useAuth.ts    /api/auth/me を呼んで現在ユーザー取得
├── lib/           # ライブラリ初期化
│   └── queryClient.ts  TanStack Query 設定
├── routes/        # 各画面 (ページコンポーネント)
│   ├── Login.tsx
│   ├── Register.tsx
│   ├── Timeline.tsx
│   ├── Groups.tsx
│   ├── GroupNew.tsx
│   ├── GroupDetail.tsx
│   ├── PostNew.tsx
│   └── InviteAccept.tsx
├── App.tsx        # ルーティング定義
└── main.tsx       # エントリ (QueryClientProvider など)
```

## ルーティング

| Path | 認証 | 画面 |
| --- | --- | --- |
| /login | 不要 | ログイン |
| /register | 不要 | 新規登録 |
| /invite/:token | 不要 | 招待リンクプレビュー + 参加 |
| / | 必要 | タイムライン (ホーム) |
| /groups | 必要 | グループ一覧 |
| /groups/new | 必要 | グループ作成 |
| /groups/:id | 必要 | グループ詳細 (メンバー・投稿・招待発行) |
| /groups/:id/posts/new | 必要 | 投稿作成 |

## 主要ライブラリ
- **React Router v6** - クライアントサイドルーティング
- **TanStack Query (React Query)** - API呼び出しのキャッシュ・再取得・状態管理
- **Tailwind CSS v4** - スタイリング

## API 呼び出しパターン

```tsx
// 取得 (useQuery)
const { data, isLoading, error } = useQuery({
  queryKey: ["timeline"],
  queryFn: () => api.get<Paginated<Post>>("/api/timeline"),
});

// 更新 (useMutation)
const mutation = useMutation({
  mutationFn: () => api.post("/api/groups", { name, display_name }),
  onSuccess: (data) => navigate(`/groups/${data.id}`),
});
```

## Cookie 認証
- `api.client` で `credentials: "include"` を必ず指定 → セッションCookie送信
- Vite proxy が `/api` を `localhost:8080` に転送 → 同一オリジン扱いで Cookie が通る

## エラーハンドリング
- `ApiError` クラスで HTTP ステータスとメッセージ保持
- 401 受信時は `useAuth` が `null` を返す → `ProtectedRoute` が `/login` リダイレクト

## 将来追加する画面 (未実装)
- 設定 (通知ON/OFF、メールアドレス変更)
- 投稿詳細 (返信機能と一緒に)
- 投稿削除UI (現状はAPIだけ)
- 誕生日豪華バナー (タイムライン上部)
- グループ別プロフィール編集
