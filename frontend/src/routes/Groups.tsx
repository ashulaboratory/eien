import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import type { Group, Paginated } from "../api/types";

// Groups は自分が所属するグループ一覧画面のコンポーネント。
// GET /api/groups で自分が入っている全グループを取得し、リスト表示する。
// 各グループ項目をクリックすると /groups/:id (詳細画面) に遷移。
export function Groups() {
  // 所属グループ一覧を取得
  const { data, isLoading, error } = useQuery({
    queryKey: ["groups"],
    queryFn: () => api.get<Paginated<Group>>("/api/groups"),
  });

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-bold">グループ</h2>
        <Link
          to="/groups/new"
          className="text-sm bg-blue-600 text-white px-3 py-1 rounded hover:bg-blue-700"
        >
          + 新規作成
        </Link>
      </div>

      {/* ローディング / エラー */}
      {isLoading && <p className="text-gray-500">読み込み中…</p>}
      {error && <p className="text-red-600">エラー: {(error as Error).message}</p>}

      {/* グループ0件の時 */}
      {data?.items?.length === 0 && (
        <div className="bg-white rounded-lg p-8 text-center text-gray-500">
          まだグループに参加していません。
          <br />
          新規作成するか、招待リンクから参加してください。
        </div>
      )}

      {/* グループ一覧: 各グループをカード形式で表示 */}
      {data?.items?.map((g) => (
        <Link
          key={g.id}
          to={`/groups/${g.id}`}
          className="block bg-white rounded-lg shadow p-4 hover:shadow-md transition"
        >
          <div className="flex justify-between items-start">
            <div>
              <h3 className="font-bold text-lg">{g.name}</h3>
              {g.description && (
                <p className="text-sm text-gray-600 mt-1">{g.description}</p>
              )}
              {/* このグループでの自分の表示名 + 管理者バッジ */}
              <p className="text-xs text-gray-500 mt-2">
                あなた: <span className="font-medium">{g.my_display_name}</span>
                {g.my_role === "admin" && <span className="ml-1 text-blue-600">(管理者)</span>}
              </p>
            </div>
          </div>
        </Link>
      ))}
    </div>
  );
}
