import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import { useTaskTypes } from "../hooks/queries";

// Generic task creation: choose a registered flow (task type) and fill in the
// context fields declared by that flow.
export function NewTaskPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const types = useTaskTypes();

  const [typeKey, setTypeKey] = useState("");
  const [title, setTitle] = useState("");
  const [intent, setIntent] = useState("");
  const [ctx, setCtx] = useState<Record<string, string>>({});
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const selected = useMemo(
    () => types.data?.find((t) => t.key === typeKey),
    [types.data, typeKey],
  );

  // Default to the first task type once loaded.
  useEffect(() => {
    if (!typeKey && types.data && types.data.length) setTypeKey(types.data[0].key);
  }, [types.data, typeKey]);

  const fields = selected ? Object.entries(selected.context_schema || {}) : [];

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      const context: Record<string, unknown> = {};
      for (const [name, t] of fields) {
        const raw = ctx[name] ?? "";
        context[name] =
          t === "number" ? Number(raw) : t === "boolean" ? raw === "true" : raw;
      }
      const detail = await api.createTask({ type: typeKey, title, intent, context });
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
      {types.data && types.data.length === 0 && (
        <div className="panel">
          <div className="empty">
            フローがありません。まず<Link to="/flows/new">フローを登録</Link>してください。
          </div>
        </div>
      )}
      {types.data && types.data.length > 0 && (
        <form className="panel" onSubmit={submit} style={{ maxWidth: 600 }}>
          <label>フロー（タスク種別）</label>
          <select value={typeKey} onChange={(e) => setTypeKey(e.target.value)}>
            {types.data.map((t) => (
              <option key={t.key} value={t.key}>
                {t.display_name}（{t.key}）
              </option>
            ))}
          </select>

          <label>題名</label>
          <input value={title} onChange={(e) => setTitle(e.target.value)} required />

          {fields.map(([name, t]) => (
            <div key={name}>
              <label>
                {name} <span className="muted">({t})</span>
              </label>
              {t === "boolean" ? (
                <select value={ctx[name] ?? "false"} onChange={(e) => setCtx((c) => ({ ...c, [name]: e.target.value }))}>
                  <option value="false">false</option>
                  <option value="true">true</option>
                </select>
              ) : (
                <input
                  type={t === "number" ? "number" : "text"}
                  value={ctx[name] ?? ""}
                  onChange={(e) => setCtx((c) => ({ ...c, [name]: e.target.value }))}
                />
              )}
            </div>
          ))}

          <label>意図（任意）</label>
          <textarea value={intent} onChange={(e) => setIntent(e.target.value)} rows={2} />

          {err && <p className="err">{err}</p>}
          <div style={{ marginTop: 16 }}>
            <button className="btn-primary" disabled={busy}>
              {busy ? "作成中…" : "タスクを作成して開始"}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
