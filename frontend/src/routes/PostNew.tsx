import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";

const MAX_IMAGES = 4;
const MAX_BODY = 1000;

export function PostNew() {
  const { id } = useParams<{ id: string }>();
  const groupId = id!;
  const [body, setBody] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const navigate = useNavigate();
  const qc = useQueryClient();

  const create = useMutation({
    mutationFn: () => {
      const form = new FormData();
      form.append("body", body);
      files.forEach((f) => form.append("images", f));
      return api.post(`/api/groups/${groupId}/posts`, form);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["timeline"] });
      qc.invalidateQueries({ queryKey: ["groups", groupId, "posts"] });
      navigate(`/groups/${groupId}`);
    },
  });

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const incoming = Array.from(e.target.files ?? []);
    const merged = [...files, ...incoming].slice(0, MAX_IMAGES);
    setFiles(merged);
    e.target.value = ""; // 同じファイルを再選択できるように
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
        <textarea
          required
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={5}
          maxLength={MAX_BODY}
          placeholder="何があった？"
          className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <p className="text-xs text-gray-500 text-right">
          {body.length}/{MAX_BODY}
        </p>

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
          {files.length > 0 && (
            <div className="mt-2 grid grid-cols-4 gap-2">
              {files.map((f, i) => (
                <div key={i} className="relative">
                  <img
                    src={URL.createObjectURL(f)}
                    alt=""
                    className="w-full h-20 object-cover rounded"
                  />
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
