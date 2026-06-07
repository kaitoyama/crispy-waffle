import { useState } from "react";
import { Link } from "react-router-dom";
import { useDefinitions } from "../hooks/queries";
import { StateDiagram } from "../components/StateDiagram";
import type { WorkflowDefinition } from "../api/types";

function FlowRow({ def }: { def: WorkflowDefinition }) {
  const [open, setOpen] = useState(false);
  const stepCount = Object.keys(def.steps).length;
  return (
    <div className="panel" style={{ marginBottom: 12 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <div style={{ flex: 1 }}>
          <strong>{def.display_name}</strong>{" "}
          <span className="kv">{def.key}@{def.version}</span>
          <div className="muted" style={{ fontSize: 12 }}>
            {stepCount} ステップ · entry: {def.entry_step}
          </div>
        </div>
        <button className="btn-ghost" onClick={() => setOpen((v) => !v)}>
          {open ? "図を隠す" : "状態機械図"}
        </button>
      </div>
      {open && (
        <div style={{ marginTop: 12 }}>
          <StateDiagram def={def} />
        </div>
      )}
    </div>
  );
}

export function FlowsPage() {
  const defs = useDefinitions();
  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", marginBottom: 16 }}>
        <h1 className="page-title" style={{ margin: 0, flex: 1 }}>フロー定義</h1>
        <Link to="/flows/new" className="btn btn-primary">＋ フローを登録</Link>
      </div>
      <p className="muted" style={{ marginTop: -8 }}>
        ワークフロー（タスクが辿る有向グラフ）の一覧。登録するとそのフローでタスクを作成できます。
      </p>
      {defs.isLoading && <p className="muted">読み込み中…</p>}
      {defs.data?.map((d) => <FlowRow key={`${d.key}@${d.version}`} def={d} />)}
      {defs.data && defs.data.length === 0 && (
        <div className="panel"><div className="empty">フローがありません。</div></div>
      )}
    </div>
  );
}
