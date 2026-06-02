// アカウント画面のプレースホルダー。プロフィール編集はグループごとに行う想定。
import { useAuth } from "../hooks/useAuth";

export function Account() {
  const { data: user } = useAuth();
  return (
    <div className="py-8">
      <h1 className="font-serif text-2xl text-ink mb-4">アカウント</h1>
      {user && (
        <div className="bg-cream-soft border border-border-warm rounded-lg p-4 text-sm">
          <div className="text-ink-muted">登録メール</div>
          <div className="text-ink mt-1">{user.email}</div>
        </div>
      )}
      <p className="text-ink-muted text-sm mt-6">
        プロフィール編集や通知設定はこれから作っていく予定です。
      </p>
    </div>
  );
}
