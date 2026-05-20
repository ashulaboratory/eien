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

function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* 認証不要 */}
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/invite/:token" element={<InviteAccept />} />

        {/* 認証必須 */}
        <Route element={<ProtectedRoute />}>
          <Route element={<Layout />}>
            <Route path="/" element={<Timeline />} />
            <Route path="/groups" element={<Groups />} />
            <Route path="/groups/new" element={<GroupNew />} />
            <Route path="/groups/:id" element={<GroupDetail />} />
            <Route path="/groups/:id/posts/new" element={<PostNew />} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
