import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useTask, useTasks } from "../hooks/queries";
import { api } from "../api/client";
import { WorkflowStateView } from "../components/WorkflowStateView";
import { ActionBar } from "../components/ActionBar";
import { EventTimeline } from "../components/EventTimeline";
import { TaskGraph } from "../components/TaskGraph";

const LINK_TYPES = [
  "subtask_of",
  "blocks",
  "blocked_by",
  "relates_to",
  "triggered_by",
  "approval_for",
  "supersedes",
];

function LinkForm({ taskId }: { taskId: string }) {
  const tasks = useTasks();
  const qc = useQueryClient();
  const [target, setTarget] = useState("");
  const [type, setType] = useState("relates_to");
  const [err, setErr] = useState<string | null>(null);

  const add = async () => {
    setErr(null);
    if (!target) return;
    try {
      await api.createLink(taskId, target, type);
      qc.invalidateQueries({ queryKey: ["task", taskId] });
      setTarget("");
    } catch (e) {
      setErr((e as Error).message);
    }
  };

  const others = (tasks.data ?? []).filter((t) => t.id !== taskId);
  return (
    <div style={{ marginTop: 12 }}>
      <div className="row" style={{ alignItems: "flex-end" }}>
        <div className="col">
          <label>関連タスク</label>
          <select value={target} onChange={(e) => setTarget(e.target.value)}>
            <option value="">— 選択 —</option>
            {others.map((t) => (
              <option key={t.id} value={t.id}>{t.title}</option>
            ))}
          </select>
        </div>
        <div className="col">
          <label>リンク種別</label>
          <select value={type} onChange={(e) => setType(e.target.value)}>
            {LINK_TYPES.map((l) => <option key={l} value={l}>{l}</option>)}
          </select>
        </div>
        <button onClick={add} style={{ height: 38 }}>リンク追加</button>
      </div>
      {err && <p className="err">{err}</p>}
    </div>
  );
}

export function TaskDetailPage() {
  const { id = "" } = useParams();
  const task = useTask(id);

  if (task.isLoading) return <p className="muted">読み込み中…</p>;
  if (task.error) return <p className="err">{(task.error as Error).message}</p>;
  if (!task.data) return null;
  const d = task.data;

  return (
    <div>
      <Link to="/" className="back-link">← ダッシュボード</Link>
      <h1 className="page-title">{d.task.title}</h1>
      <p className="muted" style={{ marginTop: -8 }}>
        <span className="kv">{d.task.type}</span> · 依頼者 {d.task.requester_id} · 担当 {d.task.assignee_id}
      </p>

      <ActionBar detail={d} />

      <div className="row">
        <div className="col">
          <WorkflowStateView detail={d} />
          <div className="panel">
            <h3>コンテキスト</h3>
            <div className="context-box">{JSON.stringify(d.task.context, null, 2)}</div>
          </div>
          <div className="panel">
            <h3>関連タスク（グラフ）</h3>
            <TaskGraph detail={d} />
            <LinkForm taskId={d.task.id} />
          </div>
        </div>
        <div className="col">
          <div className="panel">
            <h3>イベント台帳（タイムライン）</h3>
            <EventTimeline events={d.events} />
          </div>
        </div>
      </div>
    </div>
  );
}
