import { Link, useNavigate } from "react-router-dom";
import { useTasks } from "../hooks/queries";
import { StatusBadge } from "../components/badges";

export function DashboardPage() {
  const tasks = useTasks();
  const navigate = useNavigate();

  return (
    <div>
      <h1 className="page-title">ダッシュボード</h1>
      <div className="panel">
        <h3>タスク一覧</h3>
        {tasks.isLoading && <p className="muted">読み込み中…</p>}
        {tasks.error && <p className="err">{(tasks.error as Error).message}</p>}
        {tasks.data && tasks.data.length === 0 && (
          <div className="empty">
            タスクがありません。<Link to="/new">新規タスクを作成</Link>してください。
          </div>
        )}
        {tasks.data && tasks.data.length > 0 && (
          <table>
            <thead>
              <tr>
                <th>題名</th>
                <th>種別</th>
                <th>状態</th>
                <th>担当</th>
                <th>更新</th>
              </tr>
            </thead>
            <tbody>
              {tasks.data.map((t) => (
                <tr key={t.id} className="clickable" onClick={() => navigate(`/tasks/${t.id}`)}>
                  <td>{t.title}</td>
                  <td><span className="kv">{t.type}</span></td>
                  <td><StatusBadge status={t.status} /></td>
                  <td>{t.assignee_id}</td>
                  <td className="muted">{t.updated_at}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
