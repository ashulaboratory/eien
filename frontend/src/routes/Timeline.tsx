import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { Paginated, Post } from "../api/types";

export function Timeline() {
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

      {isLoading && <p className="text-gray-500">読み込み中…</p>}
      {error && <p className="text-red-600">エラー: {(error as Error).message}</p>}

      {data?.items?.length === 0 && (
        <div className="bg-white rounded-lg p-8 text-center text-gray-500">
          まだ投稿がありません。
          <br />
          <Link to="/groups" className="text-blue-600 hover:underline">
            グループから投稿してみましょう
          </Link>
        </div>
      )}

      {data?.items?.map((post) => (
        <article key={post.id} className="bg-white rounded-lg shadow p-4 space-y-3">
          <header className="flex justify-between items-start">
            <div className="flex items-center gap-2">
              <Avatar name={post.author.display_name} url={post.author.icon_url} />
              <div>
                <p className="text-sm font-medium">{post.author.display_name}</p>
                <p className="text-xs text-gray-500">@{post.group?.name}</p>
              </div>
            </div>
            <p className="text-xs text-gray-500">
              {new Date(post.created_at).toLocaleString("ja-JP")}
            </p>
          </header>

          <p className="text-gray-800 whitespace-pre-wrap">{post.body}</p>

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
