import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";

// Creates an expense.transport task. The backend starts the run and advances to
// the first wait, so the new task lands on its detail page ready for action.
export function NewTaskPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [title, setTitle] = useState("交通費の精算");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [intent, setIntent] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      const detail = await api.createTask({
        type: "expense.transport",
        title,
        intent,
        context: { route_from: from, route_to: to },
      });
      qc.invalidateQueries({ queryKey: ["tasks"] });
      navigate(`/tasks/${detail.task.id}`);
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <h1 className="page-title">新規タスク</h1>
      <form className="panel" onSubmit={submit} style={{ maxWidth: 560 }}>
        <h3>交通費の精算（expense.transport）</h3>
        <label>題名</label>
        <input value={title} onChange={(e) => setTitle(e.target.value)} required />
        <div className="row">
          <div className="col">
            <label>出発駅</label>
            <input value={from} onChange={(e) => setFrom(e.target.value)} placeholder="北千住" required />
          </div>
          <div className="col">
            <label>到着駅</label>
            <input value={to} onChange={(e) => setTo(e.target.value)} placeholder="御茶ノ水" required />
          </div>
        </div>
        <label>意図（任意）</label>
        <textarea value={intent} onChange={(e) => setIntent(e.target.value)} rows={2} />
        {err && <p className="err">{err}</p>}
        <div style={{ marginTop: 16 }}>
          <button className="btn-primary" disabled={busy}>
            {busy ? "作成中…" : "タスクを作成して開始"}
          </button>
        </div>
      </form>
    </div>
  );
}
