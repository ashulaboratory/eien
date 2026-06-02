// アプリ全体のルーティング定義。
// React Router でネスト構造を使い、認証チェック・共通レイアウトを親ルートに集約する。
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { Login } from "./routes/Login";
import { Register } from "./routes/Register";
import { Timeline } from "./routes/Timeline";
import { Groups } from "./routes/Groups";
import { GroupNew } from "./routes/GroupNew";
import { GroupDetail } from "./routes/GroupDetail";
import { PostNew } from "./routes/PostNew";
import { InviteAccept } from "./routes/InviteAccept";
import { Chat } from "./routes/Chat";
import { Account } from "./routes/Account";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* 認証不要のルート: ログイン前でもアクセスできる */}
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/invite/:token" element={<InviteAccept />} />

        {/* 認証必須のルート: ProtectedRoute で認証チェック → Layout で共通UI → 各ページ */}
        <Route element={<ProtectedRoute />}>
          {/* 親ルートに ProtectedRoute を置くと、すべての子ルートに認証が適用される */}
          <Route element={<Layout />}>
            {/* さらに Layout で共通の header/footer/nav を全ページに適用 */}
            <Route path="/" element={<Timeline />} />
            <Route path="/chat" element={<Chat />} />
            {/* /posts/new は多対多モデルの新規マイルストーン作成画面 (グループ非依存) */}
            <Route path="/posts/new" element={<PostNew />} />
            <Route path="/groups" element={<Groups />} />
            <Route path="/groups/new" element={<GroupNew />} />
            <Route path="/groups/:id" element={<GroupDetail />} />
            <Route path="/account" element={<Account />} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
