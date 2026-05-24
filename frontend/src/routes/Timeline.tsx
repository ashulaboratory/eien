import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { Paginated, Post } from "../api/types";

// Timeline はホーム画面のコンポーネント。
// 自分が所属する全グループの投稿を新しい順に表示する。
// GET /api/timeline を呼んで結果を表示。
export function Timeline() {
  // useQuery で /api/timeline からタイムラインデータを取得 (キャッシュ管理込み)
  const { data, isLoading, error } = useQuery({
    queryKey: ["timeline"],
    queryFn: () => api.get<Paginated<Post>>("/api/timeline"),
  });

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-bold">タイムライン</h2>
        <Link
          to="/groups"
          className="text-sm bg-blue-600 text-white px-3 py-1 rounded hover:bg-blue-700"
        >
          投稿する
        </Link>
      </div>

      {/* ローディング / エラー / 空状態の表示 */}
      {isLoading && <p className="text-gray-500">読み込み中…</p>}
      {error && <p className="text-red-600">エラー: {(error as Error).message}</p>}

      {/* 投稿が0件の時 */}
      {data?.items?.length === 0 && (
        <div className="bg-white rounded-lg p-8 text-center text-gray-500">
          まだ投稿がありません。
          <br />
          <Link to="/groups" className="text-blue-600 hover:underline">
            グループから投稿してみましょう
          </Link>
        </div>
      )}

      {/* 投稿リスト: map で配列を描画 (key には post.id を指定) */}
      {data?.items?.map((post) => (
        <article key={post.id} className="bg-white rounded-lg shadow p-4 space-y-3">
          <header className="flex justify-between items-start">
            <div className="flex items-center gap-2">
              {/* アバター (画像があれば画像、なければ頭文字) */}
              <Avatar name={post.author.display_name} url={post.author.icon_url} />
              <div>
                {/* グループ内の表示名 (Eien 固有: グループごとに別名が可能) */}
                <p className="text-sm font-medium">{post.author.display_name}</p>
                <p className="text-xs text-gray-500">@{post.group?.name}</p>
              </div>
            </div>
            <p className="text-xs text-gray-500">
              {new Date(post.created_at).toLocaleString("ja-JP")}
            </p>
          </header>

          {/* 投稿本文 (whitespace-pre-wrap で改行を保持) */}
          <p className="text-gray-800 whitespace-pre-wrap">{post.body}</p>

          {/* 画像 (1枚なら横幅いっぱい、複数なら 2x のグリッド) */}
          {post.images?.length > 0 && (
            <div className={`grid gap-2 ${post.images.length === 1 ? "grid-cols-1" : "grid-cols-2"}`}>
              {post.images.map((url, i) => (
                <img key={i} src={url} alt="" className="rounded w-full object-cover max-h-80" />
              ))}
            </div>
          )}
        </article>
      ))}
    </div>
  );
}

// Avatar は投稿者のアバターを描画する小コンポーネント。
// icon_url があれば画像、なければ表示名の頭文字を丸い色背景で表示。
function Avatar({ name, url }: { name: string; url?: string }) {
  if (url) {
    return <img src={url} alt={name} className="w-8 h-8 rounded-full object-cover" />;
  }
  return (
    <div className="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center text-blue-700 font-bold text-sm">
      {name.charAt(0)}
    </div>
  );
}
