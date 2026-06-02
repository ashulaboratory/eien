// PostNew は新規マイルストーン作成画面 (多対多モデル)。
// 1 つのマイルストーンを 0〜N グループに公開できる。
// 公開先 0 個も許可 = 自分専用の記録として残せる。
//
// 既存仕様との違い:
// - URL は /posts/new (グループID不要)
// - 公開先はチェックボックスのリストで複数選択
// - ?group_id=xxx クエリで初期チェック (タイムラインのフィルタチップから来た時のショートカット)
import { useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Group, Paginated } from "../api/types";

// バックエンドの定義と揃える
const MAX_IMAGES = 4;
const MAX_BODY = 1000;

export function PostNew() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [searchParams] = useSearchParams();

  // 入力 state
  const [body, setBody] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  // 公開先グループIDの集合 (Set にすることで add/delete が O(1) になる)
  // 初期値: ?group_id=xxx があればそれだけ入れる、無ければ空
  const [selected, setSelected] = useState<Set<string>>(() => {
    const init = new Set<string>();
    const gid = searchParams.get("group_id");
    if (gid) init.add(gid);
    return init;
  });

  // 自分が所属するグループ一覧を取得
  const groupsQuery = useQuery({
    queryKey: ["groups"],
    queryFn: () => api.get<Paginated<Group>>("/api/groups"),
  });

  // ボタン下のサマリー (公開先0個 = 自分専用の文言を出す)
  const summary = useMemo(() => {
    if (selected.size === 0) {
      return "公開先なし（自分専用の記録として保存されます）";
    }
    return `${selected.size} グループに公開`;
  }, [selected]);

  // マイルストーン作成 (multipart/form-data)
  const create = useMutation({
    mutationFn: () => {
      const form = new FormData();
      form.append("body", body);
      files.forEach((f) => form.append("images", f));
      // group_ids はカンマ区切り (バックエンドの parseGroupIDs が受け取る形式)
      if (selected.size > 0) {
        form.append("group_ids", Array.from(selected).join(","));
      }
      return api.post("/api/posts", form);
    },
    onSuccess: () => {
      // タイムラインキャッシュを無効化して最新を取得させる
      qc.invalidateQueries({ queryKey: ["timeline"] });
      navigate("/");
    },
  });

  // ファイル選択 (既存と新規をマージ + 上限切り詰め)
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const incoming = Array.from(e.target.files ?? []);
    const merged = [...files, ...incoming].slice(0, MAX_IMAGES);
    setFiles(merged);
    e.target.value = "";
  };

  // グループのチェック切替
  const toggleGroup = (gid: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(gid)) next.delete(gid);
      else next.add(gid);
      return next;
    });
  };

  return (
    <div className="space-y-5">
      <div className="flex items-center gap-2">
        <Link to="/" className="text-terracotta text-sm hover:underline">
          ← ホーム
        </Link>
        <h1 className="font-serif text-2xl text-ink">マイルストーンを書く</h1>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate();
        }}
        className="bg-cream-soft border border-border-warm rounded-2xl p-5 space-y-4"
      >
        {/* 本文 */}
        <div>
          <textarea
            required
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={5}
            maxLength={MAX_BODY}
            placeholder="今日のマイルストーンを残そう。"
            className="
              w-full px-3 py-2 bg-cream border border-border-warm rounded-lg
              focus:outline-none focus:ring-2 focus:ring-honey
              text-ink placeholder:text-ink-muted/60
            "
          />
          <p className="text-xs text-ink-muted text-right mt-1">
            {body.length}/{MAX_BODY}
          </p>
        </div>

        {/* 画像 */}
        <div>
          <label className="block text-sm text-ink-muted mb-1">
            画像 ({files.length}/{MAX_IMAGES})
          </label>
          <input
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            multiple
            onChange={handleFileChange}
            disabled={files.length >= MAX_IMAGES}
            className="text-sm text-ink-muted"
          />
          {files.length > 0 && (
            <div className="mt-3 grid grid-cols-4 gap-2">
              {files.map((f, i) => (
                <div key={i} className="relative">
                  <img
                    src={URL.createObjectURL(f)}
                    alt=""
                    className="w-full h-20 object-cover rounded border border-border-warm"
                  />
                  <button
                    type="button"
                    onClick={() =>
                      setFiles(files.filter((_, idx) => idx !== i))
                    }
                    className="
                      absolute top-1 right-1 bg-ink/70 text-cream-soft
                      rounded-full w-5 h-5 text-xs flex items-center justify-center
                    "
                  >
                    ×
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* 公開先グループ (チェックボックスリスト、デフォルトは全部 OFF) */}
        <div>
          <label className="block text-sm text-ink-muted mb-2">
            公開先グループ（選ばなければ自分専用）
          </label>
          {groupsQuery.isLoading && (
            <p className="text-sm text-ink-muted">グループを読み込み中…</p>
          )}
          {groupsQuery.data?.items?.length === 0 && (
            <p className="text-sm text-ink-muted">
              まだグループがありません。
              <Link to="/groups/new" className="text-terracotta underline ml-1">
                作る
              </Link>
            </p>
          )}
          <div className="space-y-2">
            {groupsQuery.data?.items?.map((g) => (
              <label
                key={g.id}
                className="
                  flex items-center gap-3 px-3 py-2 bg-cream border border-border-warm
                  rounded-lg cursor-pointer hover:border-honey transition-colors
                "
              >
                <input
                  type="checkbox"
                  checked={selected.has(g.id)}
                  onChange={() => toggleGroup(g.id)}
                  className="accent-terracotta w-4 h-4"
                />
                <div className="flex-1">
                  <div className="text-sm text-ink">{g.name}</div>
                  {g.my_display_name && (
                    <div className="text-xs text-ink-muted">
                      表示名: {g.my_display_name}
                    </div>
                  )}
                </div>
              </label>
            ))}
          </div>
        </div>

        {/* サマリー + 送信 */}
        <p className="text-xs text-ink-muted">{summary}</p>

        {create.error && (
          <p className="text-rose text-sm">
            {(create.error as Error).message}
          </p>
        )}

        <button
          type="submit"
          disabled={create.isPending || !body}
          className="
            w-full bg-terracotta text-cream-soft py-2.5 rounded-full font-medium
            hover:bg-terracotta-dark transition-colors
            disabled:opacity-50 disabled:cursor-not-allowed
          "
        >
          {create.isPending ? "保存中…" : "マイルストーンを残す"}
        </button>
      </form>
    </div>
  );
}
