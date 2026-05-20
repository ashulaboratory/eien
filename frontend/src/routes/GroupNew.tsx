import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import type { Group } from "../api/types";

export function GroupNew() {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [displayName, setDisplayName] = useState("");
  const navigate = useNavigate();
  const qc = useQueryClient();

  const create = useMutation({
    mutationFn: () =>
      api.post<Group>("/api/groups", {
        name,
        description,
        display_name: displayName,
      }),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["groups"] });
      navigate(`/groups/${data.id}`);
    },
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Link to="/groups" className="text-blue-600 text-sm">← 戻る</Link>
        <h2 className="text-xl font-bold">グループを作成</h2>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate();
        }}
        className="bg-white rounded-lg shadow p-6 space-y-4"
      >
        <div>
          <label className="block text-sm text-gray-700 mb-1">グループ名</label>
          <input
            type="text"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="例: 高校時代の仲間"
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-700 mb-1">説明（任意）</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="block text-sm text-gray-700 mb-1">
            このグループでのあなたの表示名
          </label>
          <input
            type="text"
            required
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="例: たかし"
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p className="text-xs text-gray-500 mt-1">グループごとに表示名を変えられます</p>
        </div>

        {create.error && (
          <p className="text-red-600 text-sm">{(create.error as Error).message}</p>
        )}

        <button
          type="submit"
          disabled={create.isPending}
          className="w-full bg-blue-600 text-white py-2 rounded font-medium hover:bg-blue-700 disabled:opacity-50"
        >
          {create.isPending ? "作成中…" : "作成"}
        </button>
      </form>
    </div>
  );
}
