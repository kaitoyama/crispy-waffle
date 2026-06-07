import { Link, Route, Routes, useLocation } from "react-router-dom";
import { ActorSwitcher } from "./components/ActorSwitcher";
import { DashboardPage } from "./pages/DashboardPage";
import { TaskDetailPage } from "./pages/TaskDetailPage";
import { ApprovalsInboxPage } from "./pages/ApprovalsInboxPage";
import { NewTaskPage } from "./pages/NewTaskPage";
import { FlowsPage } from "./pages/FlowsPage";
import { FlowBuilderPage } from "./pages/FlowBuilderPage";
import { ToolsPage } from "./pages/ToolsPage";
import { useApprovals } from "./hooks/queries";

function NavBar() {
  const loc = useLocation();
  const approvals = useApprovals();
  const pending = approvals.data?.length ?? 0;
  const is = (p: string) =>
    loc.pathname === p || (p !== "/" && loc.pathname.startsWith(p));
  return (
    <header className="topbar">
      <div className="brand">
        <span className="logo">◆</span> crispy-waffle
        <span className="tagline">タスクグラフ・ワークフロー</span>
      </div>
      <nav>
        <Link className={is("/") && loc.pathname === "/" ? "active" : ""} to="/">
          ダッシュボード
        </Link>
        <Link className={is("/approvals") ? "active" : ""} to="/approvals">
          承認インボックス{pending > 0 && <span className="badge-count">{pending}</span>}
        </Link>
        <Link className={is("/flows") ? "active" : ""} to="/flows">
          フロー
        </Link>
        <Link className={is("/tools") ? "active" : ""} to="/tools">
          ツール
        </Link>
        <Link className={is("/new") ? "active" : ""} to="/new">
          ＋ 新規タスク
        </Link>
      </nav>
      <ActorSwitcher />
    </header>
  );
}

export default function App() {
  return (
    <div className="app">
      <NavBar />
      <main className="content">
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/new" element={<NewTaskPage />} />
          <Route path="/tasks/:id" element={<TaskDetailPage />} />
          <Route path="/approvals" element={<ApprovalsInboxPage />} />
          <Route path="/flows" element={<FlowsPage />} />
          <Route path="/flows/new" element={<FlowBuilderPage />} />
          <Route path="/tools" element={<ToolsPage />} />
        </Routes>
      </main>
    </div>
  );
}
