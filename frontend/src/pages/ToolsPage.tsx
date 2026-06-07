import { useTools } from "../hooks/queries";
import type { ToolField } from "../api/types";

const SEC_BADGE: Record<string, string> = {
  read_only: "green",
  reversible_write: "blue",
  irreversible_or_monetary: "red",
};

function Fields({ label, fields }: { label: string; fields?: ToolField[] }) {
  if (!fields || fields.length === 0) return null;
  return (
    <div style={{ fontSize: 12, marginTop: 4 }}>
      <span className="muted">{label}: </span>
      {fields.map((f, i) => (
        <span key={f.name} className="kv">
          {i > 0 && ", "}
          {f.name}
          <span className="muted">:{f.type}</span>
          {f.required && <span style={{ color: "var(--amber)" }}>*</span>}
        </span>
      ))}
    </div>
  );
}

// Read-only catalog of the tools available to flows. Tools are authored in code
// (internal/tools/builtin); this page reflects whatever is registered.
export function ToolsPage() {
  const tools = useTools();
  return (
    <div>
      <h1 className="page-title">ツールカタログ</h1>
      <p className="muted" style={{ marginTop: -8 }}>
        ステップに束縛できる能力（ツール）の一覧。ツールはコードで追加します
        （<span className="kv">internal/tools/builtin</span> に実装し1行登録）。
      </p>
      {tools.isLoading && <p className="muted">読み込み中…</p>}
      <div className="panel">
        {tools.data?.map((t) => (
          <div key={t.key} className="stepcard" style={{ display: "block" }}>
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <strong>{t.display_name}</strong>
              <span className="kv">{t.key}</span>
              <span className={`badge ${SEC_BADGE[t.side_effect_class] ?? "gray"}`} style={{ marginLeft: "auto" }}>
                {t.side_effect_class}
              </span>
            </div>
            {t.description && <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>{t.description}</div>}
            <Fields label="入力" fields={t.inputs} />
            <Fields label="出力" fields={t.outputs} />
          </div>
        ))}
      </div>
    </div>
  );
}
