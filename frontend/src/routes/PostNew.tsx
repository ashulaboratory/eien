import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";

// 投稿に関する制約値 (バックエンドの定義と揃える)
const MAX_IMAGES = 4;   // 1投稿あたりの最大画像数
const MAX_BODY = 1000;  // 投稿本文の最大文字数

// PostNew は新規投稿画面のコンポーネント。
// 本文 + 画像 (最大4枚) を multipart/form-data で送信する。
// 成功時はタイムラインとグループ投稿一覧のキャッシュを無効化してグループへ戻る。
export function PostNew() {
  // URL パラメータからグループIDを取得
  const { id } = useParams<{ id: string }>();
  const groupId = id!;
  const [body, setBody] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const navigate = useNavigate();
  const qc = useQueryClient();

  // 投稿作成処理 (multipart/form-data で送信)
  const create = useMutation({
    mutationFn: () => {
      // FormData にテキストと画像をまとめる
      const form = new FormData();
      form.append("body", body);
      files.forEach((f) => form.append("images", f));
      // api.post は FormData を検知して JSON 化せずそのまま送る
      return api.post(`/api/groups/${groupId}/posts`, form);
    },
    onSuccess: () => {
      // 関連キャッシュを無効化 → 次回参照時に最新を取得
      qc.invalidateQueries({ queryKey: ["timeline"] });
      qc.invalidateQueries({ queryKey: ["groups", groupId, "posts"] });
      // グループ詳細に戻る
      navigate(`/groups/${groupId}`);
    },
  });

  // ファイル選択時のハンドラ
  // 既存ファイルと新規ファイルをマージし、最大数で切り詰める
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const incoming = Array.from(e.target.files ?? []);
    const merged = [...files, ...incoming].slice(0, MAX_IMAGES);
    setFiles(merged);
    e.target.value = ""; // 同じファイルを再選択できるようにクリア
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Link to={`/groups/${groupId}`} className="text-blue-600 text-sm">← グループへ</Link>
        <h2 className="text-xl font-bold">投稿</h2>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate();
        }}
        className="bg-white rounded-lg shadow p-4 space-y-3"
      >
        {/* 本文入力 */}
        <textarea
          required
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={5}
          maxLength={MAX_BODY}
          placeholder="何があった?"
          className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <p className="text-xs text-gray-500 text-right">
          {body.length}/{MAX_BODY}
        </p>

        {/* 画像選択 + プレビュー */}
        <div>
          <label className="block text-sm text-gray-700 mb-1">
            画像 ({files.length}/{MAX_IMAGES})
          </label>
          <input
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            multiple
            onChange={handleFileChange}
            disabled={files.length >= MAX_IMAGES}
            className="text-sm"
          />
          {/* 選択した画像のプレビュー (URL.createObjectURL でローカル画像表示) */}
          {files.length > 0 && (
            <div className="mt-2 grid grid-cols-4 gap-2">
              {files.map((f, i) => (
                <div key={i} className="relative">
                  <img
                    src={URL.createObjectURL(f)}
                    alt=""
                    className="w-full h-20 object-cover rounded"
                  />
                  {/* 削除ボタン: クリックでその画像を配列から除外 */}
                  <button
                    type="button"
                    onClick={() => setFiles(files.filter((_, idx) => idx !== i))}
                    className="absolute top-1 right-1 bg-black/50 text-white rounded-full w-5 h-5 text-xs flex items-center justify-center"
                  >
                    ×
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {create.error && (
          <p className="text-red-600 text-sm">{(create.error as Error).message}</p>
        )}

        <button
          type="submit"
          disabled={create.isPending || !body}
          className="w-full bg-blue-600 text-white py-2 rounded font-medium hover:bg-blue-700 disabled:opacity-50"
        >
          {create.isPending ? "投稿中…" : "投稿する"}
        </button>
      </form>
    </div>
  );
}
