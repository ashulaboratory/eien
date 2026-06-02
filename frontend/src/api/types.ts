// バックエンド API のレスポンス型定義。
// ハンドラ側の JSON 構造と1対1で対応する。
// 型を集約することで、useQuery<User>() のように呼び出し側で型推論が効くようになる。

// User: 認証中のユーザー情報 (GET /api/auth/me、register、login のレスポンス)
export interface User {
  id: string;
  email: string;
  birthday?: string; // YYYY-MM-DD 形式
}

// Group: グループ情報 (詳細表示 + 自分のメンバーシップ情報を含む)
export interface Group {
  id: string;
  name: string;
  description: string;
  created_at?: string;
  my_display_name?: string;       // このグループでの自分の表示名
  my_icon_url?: string;
  my_role?: "admin" | "member";   // role は2値の文字列リテラル型
  joined_at?: string;
}

// GroupMember: グループ内のメンバー情報
export interface GroupMember {
  user_id: string;
  display_name: string;  // グループ単位の表示名 (Eien 固有の設計)
  icon_url: string;
  role: "admin" | "member";
  joined_at: string;
}

// InvitePreview: 招待リンクのプレビュー (GET /api/invites/:token)
export interface InvitePreview {
  token: string;
  group_name: string;
  expires_at: string;
  remaining_uses: number; // max_uses - current_uses
}

// InviteCreated: 招待リンク発行時のレスポンス
export interface InviteCreated {
  token: string;
  group_id: string;
  expires_at: string;
  max_uses: number;
}

// PostAuthor: マイルストーンの著者情報 (display_name は文脈によって表示元が変わる)
//   - 統合タイムライン: 最初にシェアされたグループでの著者名 (0シェアなら最初に入ったグループの名)
//   - 特定グループ表示: そのグループでの著者名
//   - icon_url は今は backend が返していないので optional
export interface PostAuthor {
  user_id: string;
  display_name: string;
  icon_url?: string;
}

// Post: マイルストーン1件分。
// - group は廃止 (post_shares 多対多に変更)
// - share_count はそのマイルストーンが今シェアされているグループ数
export interface Post {
  id: string;
  body: string;
  images: string[];          // 画像URLの配列
  created_at: string;
  share_count?: number;      // 統合タイムライン / マイ記録ビューでのみ含まれる
  author: PostAuthor;
}

// Paginated<T>: ページネーション付きレスポンスの汎用型
// 例: Paginated<Post> でタイムラインや投稿一覧を表現
export interface Paginated<T> {
  items: T[];
  pagination?: { limit: number; offset: number };
}
