// バックエンドAPIのレスポンス型定義

export interface User {
  id: string;
  email: string;
  birthday?: string;
}

export interface Group {
  id: string;
  name: string;
  description: string;
  created_at?: string;
  my_display_name?: string;
  my_icon_url?: string;
  my_role?: "admin" | "member";
  joined_at?: string;
}

export interface GroupMember {
  user_id: string;
  display_name: string;
  icon_url: string;
  role: "admin" | "member";
  joined_at: string;
}

export interface InvitePreview {
  token: string;
  group_name: string;
  expires_at: string;
  remaining_uses: number;
}

export interface InviteCreated {
  token: string;
  group_id: string;
  expires_at: string;
  max_uses: number;
}

export interface PostAuthor {
  user_id: string;
  display_name: string;
  icon_url: string;
}

export interface Post {
  id: string;
  body: string;
  images: string[];
  created_at: string;
  group?: { id: string; name: string };
  author: PostAuthor;
}

export interface Paginated<T> {
  items: T[];
  pagination?: { limit: number; offset: number };
}
