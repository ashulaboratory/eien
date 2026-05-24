import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Group, GroupMember, InviteCreated, Paginated, Post } from "../api/types";

// GroupDetail はグループ詳細画面のコンポーネント。
// グループ情報・メンバー一覧・投稿一覧を3つの useQuery で並列取得する。
// 招待リンク発行ボタンと投稿ボタンも表示する。
export function GroupDetail() {
  // URL パラメータからグループIDを取得 (useParams のジェネリクスで型推論)
  const { id } = useParams<{ id: string }>();
  const groupId = id!;

  // ① グループ情報を取得
  const group = useQuery({
    queryKey: ["groups", groupId],
    queryFn: () => api.get<Group>(`/api/groups/${groupId}`),
  });

  // ② このグループのメンバー一覧を取得
  const members = useQuery({
    queryKey: ["groups", groupId, "members"],
    queryFn: () => api.get<Paginated<GroupMember>>(`/api/groups/${groupId}/members`),
  });

  // ③ このグループの投稿一覧を取得
  const posts = useQuery({
    queryKey: ["groups", groupId, "posts"],
    queryFn: () => api.get<Paginated<Post>>(`/api/groups/${groupId}/posts`),
  });

  // 招待リンク発行のローカルステート
  const [inviteToken, setInviteToken] = useState<string | null>(null);
  const invite = useMutation({
    mutationFn: () => api.post<InviteCreated>(`/api/groups/${groupId}/invites`),
    onSuccess: (data) => setInviteToken(data.token),
  });

  // ローディングとエラー処理
  if (group.isLoading) return <p className="text-gray-500">読み込み中…</p>;
  if (group.error) return <p className="text-red-600">エラー: {(group.error as Error).message}</p>;

  // 招待 URL の組み立て (現在のオリジン + /invite/<token>)
  const inviteUrl = inviteToken
    ? `${window.location.origin}/invite/${encodeURIComponent(inviteToken)}`
    : null;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Link to="/groups" className="text-blue-600 text-sm">← グループ一覧</Link>
      </div>

      {/* グループ基本情報 */}
      <div className="bg-white rounded-lg shadow p-4 space-y-2">
        <h2 className="text-xl font-bold">{group.data?.name}</h2>
        {group.data?.description && (
          <p className="text-sm text-gray-600">{group.data.description}</p>
        )}
      </div>

      {/* アクションボタン: 投稿 / 招待リンク発行 */}
      <div className="flex gap-2">
        <Link
          to={`/groups/${groupId}/posts/new`}
          className="flex-1 bg-blue-600 text-white text-center py-2 rounded hover:bg-blue-700"
        >
          + 投稿する
        </Link>
        <button
          onClick={() => invite.mutate()}
          disabled={invite.isPending}
          className="flex-1 bg-white border border-blue-600 text-blue-600 py-2 rounded hover:bg-blue-50 disabled:opacity-50"
        >
          {invite.isPending ? "発行中…" : "招待リンク発行"}
        </button>
      </div>

      {/* 招待リンク発行後の表示 (URL とコピーボタン) */}
      {inviteUrl && (
        <div className="bg-blue-50 border border-blue-200 rounded p-3 space-y-2">
          <p className="text-sm font-medium text-blue-900">
            招待リンクを発行しました(7日有効、1回限り)
          </p>
          <code className="block bg-white p-2 rounded text-xs break-all">{inviteUrl}</code>
          <button
            onClick={() => navigator.clipboard.writeText(inviteUrl)}
            className="text-sm text-blue-600 hover:underline"
          >
            📋 コピー
          </button>
        </div>
      )}

      {/* メンバー一覧 */}
      <section className="bg-white rounded-lg shadow p-4">
        <h3 className="font-bold mb-2">メンバー ({members.data?.items?.length ?? 0})</h3>
        <ul className="space-y-1">
          {members.data?.items?.map((m) => (
            <li key={m.user_id} className="text-sm flex justify-between">
              <span>{m.display_name}</span>
              <span className="text-gray-500 text-xs">
                {m.role === "admin" ? "管理者" : "メンバー"}
              </span>
            </li>
          ))}
        </ul>
      </section>

      {/* 投稿一覧 */}
      <section className="space-y-2">
        <h3 className="font-bold">投稿 ({posts.data?.items?.length ?? 0})</h3>
        {posts.data?.items?.length === 0 && (
          <p className="text-gray-500 text-sm">まだ投稿はありません。</p>
        )}
        {posts.data?.items?.map((p) => (
          <article key={p.id} className="bg-white rounded-lg shadow p-3 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="font-medium">{p.author.display_name}</span>
              <span className="text-xs text-gray-500">
                {new Date(p.created_at).toLocaleString("ja-JP")}
              </span>
            </div>
            <p className="whitespace-pre-wrap">{p.body}</p>
            {p.images?.length > 0 && (
              <div className={`grid gap-1 ${p.images.length === 1 ? "grid-cols-1" : "grid-cols-2"}`}>
                {p.images.map((url, i) => (
                  <img key={i} src={url} alt="" className="rounded w-full object-cover max-h-60" />
                ))}
              </div>
            )}
          </article>
        ))}
      </section>
    </div>
  );
}
