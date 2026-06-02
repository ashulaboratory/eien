// Timeline はホーム画面 (Warm Album)。
// 上部に Twitter 風の横スクロールフィルタチップ:
//   [すべて] [🔒 マイ記録] [グループA] [グループB] ...
// クリックで GET /api/timeline のクエリパラメータを切り替える。
//
// バックエンドの仕様:
//   - デフォルト: 統合タイムライン (自分の記録 + 所属グループへのシェア)
//   - ?filter=mine: 自分の記録のみ
//   - ?group_id=xxx: 特定グループにシェアされた記録のみ
import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Group, Paginated, Post } from "../api/types";

// フィルタ状態を1つの discriminated union で表現
// → どのモードか type で判別できるので switch 分岐が型安全になる
type Filter =
  | { type: "all" }
  | { type: "mine" }
  | { type: "group"; groupId: string; groupName: string };

export function Timeline() {
  const [filter, setFilter] = useState<Filter>({ type: "all" });

  // フィルタに応じた URL を組み立てる
  const timelineUrl = (() => {
    switch (filter.type) {
      case "all":
        return "/api/timeline";
      case "mine":
        return "/api/timeline?filter=mine";
      case "group":
        return `/api/timeline?group_id=${filter.groupId}`;
    }
  })();

  // タイムライン取得
  // queryKey にフィルタを含めることでフィルタ切替時に再フェッチされる
  const timelineQuery = useQuery({
    queryKey: ["timeline", filter],
    queryFn: () => api.get<Paginated<Post>>(timelineUrl),
  });

  // フィルタチップ用にグループ一覧も取得
  const groupsQuery = useQuery({
    queryKey: ["groups"],
    queryFn: () => api.get<Paginated<Group>>("/api/groups"),
  });

  // 投稿ボタンのリンク先: グループフィルタ中ならそのグループにチェック入った状態で遷移
  const composeLink =
    filter.type === "group"
      ? `/posts/new?group_id=${filter.groupId}`
      : "/posts/new";

  return (
    <div className="space-y-5">
      {/* ヘッダ: タイトル + 投稿CTA */}
      <div className="flex justify-between items-baseline">
        <h2 className="font-serif text-2xl text-ink tracking-tight">
          タイムライン
        </h2>
        <Link
          to={composeLink}
          className="
            text-sm bg-terracotta text-cream-soft
            px-4 py-1.5 rounded-full tracking-wide
            hover:bg-terracotta-dark transition-colors
          "
        >
          投稿する
        </Link>
      </div>

      {/* フィルタチップ (横スクロール) */}
      <div
        className="
          flex gap-2 overflow-x-auto pb-1
          -mx-4 px-4
          [scrollbar-width:none] [&::-webkit-scrollbar]:hidden
        "
      >
        <FilterChip
          label="すべて"
          active={filter.type === "all"}
          onClick={() => setFilter({ type: "all" })}
        />
        <FilterChip
          label="🔒 マイ記録"
          active={filter.type === "mine"}
          onClick={() => setFilter({ type: "mine" })}
        />
        {groupsQuery.data?.items?.map((g) => (
          <FilterChip
            key={g.id}
            label={g.name}
            active={filter.type === "group" && filter.groupId === g.id}
            onClick={() =>
              setFilter({ type: "group", groupId: g.id, groupName: g.name })
            }
          />
        ))}
      </div>

      {/* ステータス表示 */}
      {timelineQuery.isLoading && (
        <p className="text-ink-muted text-sm">読み込み中…</p>
      )}
      {timelineQuery.error && (
        <p className="text-rose text-sm">
          エラー: {(timelineQuery.error as Error).message}
        </p>
      )}

      {/* 0件の場合 */}
      {timelineQuery.data?.items?.length === 0 && (
        <div className="bg-cream-soft border border-border-warm rounded-2xl p-10 text-center">
          <p className="text-ink-muted text-sm">まだマイルストーンがありません。</p>
          <Link
            to={composeLink}
            className="
              inline-block mt-3 text-terracotta hover:text-terracotta-dark
              underline underline-offset-2 text-sm
            "
          >
            最初のマイルストーンを書く
          </Link>
        </div>
      )}

      {/* マイルストーンカード一覧 */}
      {timelineQuery.data?.items?.map((post) => (
        <article
          key={post.id}
          className="
            bg-cream-soft border border-border-warm rounded-2xl
            p-5 space-y-3
          "
        >
          {/* ヘッダ: アバター + 名前 + (share_count 表示) + 日時 */}
          <header className="flex justify-between items-start">
            <div className="flex items-center gap-3">
              <Avatar name={post.author.display_name} url={post.author.icon_url} />
              <div>
                <p className="text-sm font-medium text-ink">
                  {post.author.display_name}
                </p>
                {/* share_count: 0 のときは「自分専用」と表記 */}
                {typeof post.share_count === "number" && (
                  <p className="text-xs text-ink-muted">
                    {post.share_count === 0
                      ? "🔒 自分専用"
                      : `${post.share_count} グループに公開`}
                  </p>
                )}
              </div>
            </div>
            <time className="text-xs text-ink-muted/80">
              {formatDate(post.created_at)}
            </time>
          </header>

          {/* 本文 */}
          <p className="text-ink whitespace-pre-wrap leading-relaxed">
            {post.body}
          </p>

          {/* 画像 (印画紙風: クリーム背景 + 内側余白) */}
          {post.images?.length > 0 && (
            <div
              className={`grid gap-2 ${
                post.images.length === 1 ? "grid-cols-1" : "grid-cols-2"
              }`}
            >
              {post.images.map((url, i) => (
                <div
                  key={i}
                  className="bg-cream p-2 border border-border-warm rounded-lg"
                >
                  <img
                    src={url}
                    alt=""
                    className="rounded w-full object-cover max-h-80"
                  />
                </div>
              ))}
            </div>
          )}
        </article>
      ))}
    </div>
  );
}

// --- フィルタチップ ---
function FilterChip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={[
        "whitespace-nowrap px-4 py-1.5 rounded-full text-sm transition-colors border",
        active
          ? "bg-terracotta text-cream-soft border-terracotta"
          : "bg-cream-soft text-ink-muted border-border-warm hover:border-honey",
      ].join(" ")}
    >
      {label}
    </button>
  );
}

// --- アバター ---
function Avatar({ name, url }: { name: string; url?: string }) {
  if (url) {
    return (
      <img
        src={url}
        alt={name}
        className="w-10 h-10 rounded-full object-cover border border-border-warm"
      />
    );
  }
  return (
    <div className="w-10 h-10 rounded-full bg-honey/20 border border-honey/40 flex items-center justify-center text-honey font-serif text-lg">
      {name.charAt(0)}
    </div>
  );
}

// 日付フォーマット: 今日なら "HH:mm"、それより前なら "MM月DD日"
function formatDate(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const sameDay =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();
  if (sameDay) {
    return date.toLocaleTimeString("ja-JP", {
      hour: "2-digit",
      minute: "2-digit",
    });
  }
  return date.toLocaleDateString("ja-JP", {
    month: "long",
    day: "numeric",
  });
}
