# DB設計の判断記録

本ドキュメントは、Eien のデータベース設計（`backend/db/migrations/`）における主要な判断と、その背景にある理由を記録するものです。スキーマ定義そのものは `docs/design.md` および各マイグレーションファイルに記述されており、本ドキュメントは「なぜそうしたか」に焦点を当てます。

---

## テーブル一覧

| # | テーブル | 役割 |
|---|---|---|
| 1 | `users` | ユーザーアカウントの基本情報 |
| 2 | `sessions` | ログインセッションの管理 |
| 3 | `groups` | 招待制グループの基本情報 |
| 4 | `group_members` | ユーザーとグループの所属関係 + グループ単位の表示情報 |
| 5 | `invite_links` | 招待リンクの発行・使用回数管理 |
| 6 | `posts` | グループ宛の投稿本文 |
| 7 | `post_images` | 投稿に紐づく画像メタデータ |
| 8 | `email_logs` | 送信メールの監査ログ + 冪等性チェック |

---

## テーブル別の設計判断

### 1. `users`

- **`id UUID PRIMARY KEY`**: 連番ではなく UUID を採用。URL に出る ID を推測不能にし、将来の分散運用にも対応する。
- **`email TEXT UNIQUE NOT NULL`**: ログイン識別子。アプリ層の重複チェックでは競合状態が発生しうるため、UNIQUE 制約を DB レベルで保証する。
- **`password_hash TEXT NOT NULL`**: bcrypt の出力は約60文字だが、将来 Cost 変更やアルゴリズム移行で長さが変わる可能性があるため、固定長を避けて TEXT を採用。
- **`birthday DATE NOT NULL`**: 誕生日通知が Eien の主要機能のため必須。時刻もタイムゾーンも不要なため DATE 型（TIMESTAMPTZ にすると UTC 変換で日付がずれるリスク）。
- **通知フラグ3つ**: `birthday_self_notify` / `birthday_member_notify` は `DEFAULT TRUE`、`post_notify` は `DEFAULT FALSE`。これは **「通知最小限」というコンセプトをスキーマに直接反映**したもので、ユーザーが明示的にオンにしない限り投稿通知は来ない設計。

### 2. `sessions`

- **`id TEXT PRIMARY KEY`**（UUID ではない）: Cookie に格納する秘密値として、`crypto/rand` で生成した 256ビットのランダム値を base64 化したものを使用。UUID v4（122ビット）よりランダム性を上げ、UUID 型に縛られない柔軟性も確保。
- **`user_id ... ON DELETE CASCADE`**: ユーザー削除時に紐づくセッションも自動削除。ログイン主体が存在しないセッションは無価値なため。
- **`expires_at TIMESTAMPTZ NOT NULL`**: 認証ミドルウェアで `expires_at > NOW()` の条件で弾く。期限なしセッションを許容しないことで、流出時の被害を限定。
- **`updated_at` がない**: セッションは「作って・使って・消える」だけで更新の概念がない。
- **`idx_sessions_user_id`**: ログアウト時の `DELETE WHERE user_id = $1` や CASCADE 処理での内部検索を高速化。
- **期限切れセッションの掃除は未実装**: 認証としては `expires_at` で弾けるが、レコードは溜まり続ける。本番化に合わせて定期 DELETE バッチを追加する想定。

### 3. `groups`

- **`name TEXT NOT NULL` / `description TEXT (nullable)`**: 名前は必須、説明は任意。NULL は「未入力」を意味する（空文字との区別が必要な場合があるため NULL を採用）。
- **`created_by_user_id` の命名**: 将来このテーブルから複数のユーザー関連カラム（管理者、最終編集者など）が出る可能性を見越し、`user_id` ではなく役割を含めた自己文書化命名を採用。
- **CASCADE していない**: 作成者1人の削除でグループ全体が消えるのは Eien のコンセプトに反するため。
- **メンバー情報をこのテーブルに持たない**: 多対多関係を表現するため、`group_members` 中間テーブルに分離する RDB の標準パターンを採用。

### 4. `group_members`

- **`id UUID PRIMARY KEY` + `UNIQUE (user_id, group_id)`**（複合PK ではなく）: 他テーブルからの参照しやすさを優先。将来「グループ内の役職交代履歴」のようなテーブルが参照する場合、1カラムで参照できる方が構造がシンプル。
- **`display_name TEXT NOT NULL` を group_members に持たせる**（users ではなく）: **Eien 固有の設計**。同じユーザーがグループごとに違う表示名を持てる（例：高専クラスでは「むらかみ」、職場 OB 会では「村上太郎」）。クローズドSNSだからこそ成立する、Eien のコンセプトに沿った設計。
- **`icon_url` も同様にグループ単位**: グループごとに別アイコンを設定可能。
- **`role TEXT NOT NULL CHECK (role IN ('admin', 'member'))`**: 不正な値が DB に入るのを防ぐ。ENUM 型ではなく TEXT + CHECK を選んだのは、将来 role の種類を増やす可能性を考えた時の変更容易性のため。
- **`joined_at`**: `created_at` ではなく、メンバーシップの開始時刻という意味を込めた命名。
- **`idx_group_members_user_id` と `idx_group_members_group_id`**: 「ユーザーの所属グループ」「グループのメンバー」の両方向の検索を高速化。複合インデックスは先頭カラム以外で効かないため、別々に作成。
- **`idx_group_members_user_id` の冗長性**: UNIQUE (user_id, group_id) が自動で複合インデックスを作るため、user_id 単独検索もカバーされている可能性が高い。整理の余地あり。

### 5. `invite_links`

- **`id UUID` と `token TEXT UNIQUE` を分離**: `id` は内部参照用、`token` は外部公開（URL に入る）用と役割を分離。
- **`token` の生成**: `crypto/rand` で予測不能な文字列を生成（sessions.id と同じ思想）。URL 推測による不正アクセスを防ぐ。
- **`max_uses` / `current_uses` の二重管理**: 招待リンクが何回まで使えるか / 現状何回使われたかを管理。デフォルトは `max_uses=1`（1人向け1リンク）。
- **競合状態の防止**: 同時に上限近辺で使われた場合、`UPDATE invite_links SET current_uses = current_uses + 1 WHERE id = $1 AND current_uses < max_uses RETURNING current_uses` という条件付き UPDATE を使うことで、行ロックでアトミックに上限チェック+インクリメントを実現。
- **`expires_at NOT NULL`**: 期限なしリンクの流出リスクを排除するため必須。
- **明示的なインデックスなし**: PRIMARY KEY と UNIQUE の自動インデックスでよく使う検索（token 検索）はカバー。

### 6. `posts`

- **`group_id` と `author_user_id` の両方を保持**: 「どのグループへの投稿か」「誰が書いたか」を両方記録。Eien は「グループへの投稿」しかないため、`group_id` は NOT NULL。
- **`body TEXT NOT NULL`**: 長さ制限は DB レベルで設けない（アプリ層でバリデーション）。Twitter 風の固定制限を将来変更したくなった時のスキーマ変更コストを避ける。
- **`deleted_at TIMESTAMPTZ (nullable)` で論理削除**: 物理 DELETE せず、削除時刻を記録する。理由は以下：
  - うっかり削除からの復元が可能
  - 将来のリアクション・コメント機能で参照整合性を保てる
  - 「いつ消されたか」の監査
  - 統計の整合性（過去の投稿数が時間経過で減らない）
- **`is_deleted BOOLEAN` ではなく `deleted_at TIMESTAMPTZ`**: 「削除されたか」と「いつ削除されたか」を1カラムで表現できる情報量の差。Soft Delete のベストプラクティスとして広く採用されている。
- **論理削除の運用上の注意**: `WHERE deleted_at IS NULL` のフィルタが1箇所でも漏れると削除済み投稿が表示される事故になる。sqlc を使って各クエリで明示的に書く運用。
- **`idx_posts_group_id_created_at ON posts(group_id, created_at DESC)`**: タイムライン取得（`WHERE group_id = $1 ORDER BY created_at DESC LIMIT 20`）を、追加のソート処理なしでインデックスから直接返せる構造。**等価条件カラムを先頭・ソート対象を後ろ・DESC 方向まで一致**させる典型的な最適化。
- **`idx_posts_author_user_id`**: 将来のユーザー別投稿一覧を見越したインデックス。現状の API には対応するエンドポイントがなく、運用上は冗長な可能性あり。

### 7. `post_images`

- **`posts` から分離した別テーブル**: 1投稿に複数画像が紐づく1対多関係。`posts.image_url_1, image_url_2, ...` のように持つと、画像数上限の固定・NULL の海・メタ情報の欠如など問題が多い。RDB の正規化原則に従って分離。
- **`image_url TEXT NOT NULL`**: 画像本体はオブジェクトストレージ（開発：ローカルディスク、本番：Cloudflare R2）に保存し、DB には URL のみ持つ。DB に画像バイナリを入れると DB が肥大化し、画像と DB のスケール特性が結合してしまう問題を避けるため。
- **`ON DELETE CASCADE`**: 投稿が物理削除された場合、画像メタデータは無価値になるため自動削除。ただし posts は基本論理削除なので CASCADE が実際に発動するのは限定的。
- **`sort_order INTEGER`**: 複数画像の表示順序。`0, 1, 2, ...` と詰めて振る素朴な設計。1投稿あたりの画像数が少なく並び替え機能もない想定のためシンプルに保つ。
- **`updated_at` がない**: 画像メタデータは「作って・参照して・消える」だけで更新の概念がない。
- **`idx_post_images_post_id`**: 投稿表示時の `WHERE post_id = $1 ORDER BY sort_order` 用。1投稿の画像数が少ないため複合インデックスにせず単一で十分。
- **画像本体の孤児問題（未対応）**: DB の post_images を削除しても、オブジェクトストレージ上の画像本体は残る。本番化に合わせて、削除フローでストレージ側も削除するか、定期バッチで孤児を回収する仕組みが必要。

### 8. `email_logs`

- **役割は2つ**: 監査ログ（誰に・いつ・何を送ったか）と冪等性チェック（cron が複数回走っても2度送らない）。
- **`recipient_email TEXT NOT NULL` のスナップショット設計**: `users.email` を JOIN せず、送信時のメールアドレスを別カラムで保存。ユーザーが将来メールアドレスを変更しても、当時の送信先記録が改ざんされない。**監査ログの信頼性を保つには参照先依存ではなく凍結保存が必要**という発想。
- **`subject TEXT NOT NULL` も同様のスナップショット**: メール件名テンプレートが変わっても当時の件名が残る。
- **冪等性の実現**: メール送信前に「今日この人にこの種類のメールを `status='sent'` で記録したか」を確認するクエリを発行し、あればスキップする。これで何度 cron が走っても1日1通に制限。
- **`status TEXT NOT NULL CHECK (status IN ('pending', 'sent', 'failed'))`**: 送信処理のステートマシン。`pending` で INSERT → 送信処理 → 成功なら `sent` + `sent_at` セット、失敗なら `failed` + `error_message` セット。
- **`email_type` には CHECK 制約をかけない**: メール種類は将来増える想定（誕生日・招待・リマインダなど）なので、スキーマ変更を避けて TEXT で柔軟運用。
- **`created_at` と `sent_at` の使い分け**: 前者は送信試行を記録した時刻、後者は実際の送信完了時刻。送信失敗時には `sent_at` が NULL のままになる。
- **インデックス3つ**: `recipient_user_id`（ユーザー別履歴）、`(email_type, status)` 複合（失敗メール一覧など）、`created_at DESC`（時系列での最新取得）。

---

## 横串の設計原則

### 主キー設計

- 原則：**UUID PRIMARY KEY DEFAULT gen_random_uuid()**
  - URL に出る ID を推測不能にする
  - 将来の分散運用での衝突回避
  - アプリ側で ID を先に生成できる
- 例外：`sessions.id TEXT PRIMARY KEY`
  - Cookie に入れる秘密値で、`crypto/rand` で 256bit のランダム文字列を生成
  - UUID v4 より大きいランダム性を確保
  - UUID 型に縛られない柔軟性

### 時刻の型

- 原則：**`TIMESTAMPTZ NOT NULL DEFAULT NOW()`** で統一
  - UTC で内部保存、クライアントのタイムゾーンで表示
  - 分散ユーザーをまたぐ場合のズレを防ぐ
  - DB の `NOW()` を時刻の Single Source of Truth にすることで、アプリインスタンス間の時計ずれに依存しない
- 例外：`users.birthday DATE`
  - 日付の概念で、時刻もタイムゾーンも本来不要
  - TIMESTAMPTZ にすると UTC 変換で日付が1日ずれるリスク

### 外部キーと参照整合性

- 原則：**FK は付けるが、`ON DELETE` は明示せず `NO ACTION`**
  - 子レコードがある親の削除を阻止することで、データの安易な消失を防ぐ
  - 「縁を残す」という Eien のコンセプトに整合
- 例外的に `ON DELETE CASCADE`：
  - `sessions.user_id`：ユーザーが消えたらセッションは無価値
  - `post_images.post_id`：投稿が消えたら画像メタは無価値
- ユーザー削除フローは未実装。将来実装する際は論理削除 or 匿名化を検討。

### 論理削除

- `posts` のみ `deleted_at TIMESTAMPTZ` で論理削除を採用。
- BOOLEAN ではなく TIMESTAMPTZ：「削除されたか」と「いつ削除されたか」を1カラムで表現。
- `WHERE deleted_at IS NULL` のフィルタ漏れに注意。sqlc で各クエリに明示記述。

### インデックス設計

- PRIMARY KEY と UNIQUE 制約は自動でインデックスを作る。明示の `CREATE INDEX` は不要。
- 外部キーカラム（FK）は検索が頻発するため、必要に応じてインデックスを明示。
- 複合インデックス `(group_id, created_at DESC)`：等価条件カラムを先頭、ソート対象を後ろ、ソート方向まで一致させることで、ORDER BY 処理を不要にできる。
- 複合インデックスは先頭カラムから順にしか使えない（B-tree インデックスの性質）。

### 制約の使い分け

- **UNIQUE 制約**：一意性をアプリではなく DB に保証させる。`users.email`、`group_members(user_id, group_id)` など。競合状態への耐性を確保。
- **CHECK 制約**：列挙値が固定の場合に DB レベルで不正値をブロック。`group_members.role`、`email_logs.status`。
- **CHECK ではなく TEXT 自由運用**：将来種類が増える可能性のあるカラム（`email_logs.email_type` など）。

### 冪等性とスナップショット

- 定期実行ジョブ（cron）は複数回起動の可能性があるため、**冪等性を DB レベルで担保**する設計（`email_logs` で送信履歴をチェック）。
- 監査ログには **送信時の値そのものを凍結保存**する（`email_logs.recipient_email`、`subject`）。後で参照先データが変更されても履歴が改ざんされない。

### 自己文書化命名

- カラム名で意図を語る命名を優先：`created_by_user_id`、`author_user_id`、`recipient_user_id`、`joined_at`、`sent_at` など。
- 単純な `user_id` だと、テーブル内に複数のユーザー関連カラムが出てきた時に曖昧になるため避ける。

---

## 主要な設計判断のサマリー

| 設計判断 | 該当テーブル | 要点 |
|---|---|---|
| UUID PK の徹底（sessions だけ TEXT） | 全テーブル / sessions | 推測不能性 + 分散対応 / Cookie 用秘密値 |
| TIMESTAMPTZ で統一（birthday だけ DATE） | 全テーブル / users.birthday | 分散タイムゾーン対応 / 日付概念 |
| 多対多は中間テーブル | groups ↔ users via group_members | 配列カラムではなく専用テーブル |
| 表示名をグループ単位で持つ | group_members.display_name | Eien 固有の「グループごとに別人格」設計 |
| 論理削除（物理削除しない） | posts.deleted_at | 復元・整合性・監査・統計のため |
| 複合インデックスのカラム順 | posts(group_id, created_at DESC) | タイムライン用に等価条件+ソート方向まで一致 |
| 条件付き UPDATE で競合状態を防ぐ | invite_links (current_uses) | アプリ層チェックではなく DB の行ロックで保証 |
| 冪等性を DB クエリで担保 | email_logs | cron 複数回起動でも2度送らない |
| スナップショット設計 | email_logs.recipient_email / subject | 参照先依存せず当時の値を凍結 |
| 自己文書化命名 | created_by_user_id, joined_at, sent_at 等 | カラム名で意図を語る |

---

## 既知の改善余地

現状の実装で**まだ手が回っていない / 将来必要になる**点：

- **期限切れセッションの掃除**：`sessions.expires_at` 過ぎたレコードを削除する定期バッチが未実装。
- **ユーザー削除フロー**：機能自体が未実装。実装時は論理削除 or 匿名化のフロー設計が必要。
- **グループ削除フロー**：機能自体が未実装。実装時は CASCADE か論理削除かの判断が必要。
- **画像本体の孤児管理**：DB 削除時にオブジェクトストレージ側の画像本体を削除する処理が未実装。
- **`idx_group_members_user_id` の冗長性**：UNIQUE (user_id, group_id) の自動インデックスでカバーされている可能性が高く、整理の余地あり。
- **`idx_posts_author_user_id` の有効性**：対応するエンドポイントが現状なく、運用上有効活用されていない可能性あり。
