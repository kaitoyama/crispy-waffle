import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";
import { useTools } from "../hooks/queries";
import type { ContextFieldInput, RegisterFlowRequest, StepInput } from "../api/types";

type StepDraft = StepInput & { _id: number };

let counter = 1;
const uid = () => counter++;

const KINDS = [
  { v: "agent_action", l: "agent_action（システム/プログラム実行）" },
  { v: "human_gate", l: "human_gate（人間ゲート）" },
  { v: "system_action", l: "system_action（システム処理）" },
  { v: "sub_workflow", l: "sub_workflow（サブフロー）" },
];
const GATE_KINDS = [
  { v: "approval", l: "approval（他者の承認）" },
  { v: "elicitation", l: "elicitation（実行者の確認 Yes/No）" },
  { v: "trigger", l: "trigger（実行トリガ／ボタン）" },
];

function blankStep(key: string, kind: string): StepDraft {
  return {
    _id: uid(),
    key,
    title: "",
    kind,
    gate_kind: kind === "human_gate" ? "approval" : "",
    enters_state: "",
    terminal: false,
    tool_key: "",
    transitions: [],
  };
}

// A sensible default flow so the page is usable immediately: a single approval
// gate that branches to two terminal states.
function defaultSteps(): StepDraft[] {
  const submit = blankStep("submit", "human_gate");
  submit.title = "承認待ち";
  submit.gate_kind = "approval";
  submit.enters_state = "Pending";
  submit.transitions = [
    { to: "done", guard: 'approval_approved("submit")' },
    { to: "rejected", guard: 'approval_rejected("submit")' },
  ];
  const done = blankStep("done", "system_action");
  done.title = "承認済み";
  done.enters_state = "Approved";
  done.terminal = true;
  const rejected = blankStep("rejected", "system_action");
  rejected.title = "却下";
  rejected.enters_state = "Rejected";
  rejected.terminal = true;
  return [submit, done, rejected];
}

export function FlowBuilderPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const tools = useTools();

  const [key, setKey] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [contextFields, setContextFields] = useState<ContextFieldInput[]>([
    { name: "subject", type: "string" },
  ]);
  const [steps, setSteps] = useState<StepDraft[]>(defaultSteps);
  const [entryStep, setEntryStep] = useState("submit");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const stepKeys = steps.map((s) => s.key).filter(Boolean);

  const patchStep = (id: number, patch: Partial<StepDraft>) =>
    setSteps((prev) => prev.map((s) => (s._id === id ? { ...s, ...patch } : s)));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      const body: RegisterFlowRequest = {
        key: key.trim(),
        display_name: displayName.trim(),
        entry_step: entryStep,
        context_fields: contextFields.filter((f) => f.name.trim()),
        steps: steps.map(({ _id, ...s }) => s),
      };
      await api.registerFlow(body);
      qc.invalidateQueries({ queryKey: ["definitions"] });
      qc.invalidateQueries({ queryKey: ["taskTypes"] });
      navigate("/flows");
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <h1 className="page-title">フローを登録</h1>
      <form onSubmit={submit}>
        <div className="panel">
          <h3>基本情報</h3>
          <div className="row">
            <div className="col">
              <label>キー（例: expense.lodging）</label>
              <input value={key} onChange={(e) => setKey(e.target.value)} placeholder="expense.lodging" required />
            </div>
            <div className="col">
              <label>表示名</label>
              <input value={displayName} onChange={(e) => setDisplayName(e.target.value)} placeholder="宿泊費の精算" />
            </div>
          </div>
        </div>

        <div className="panel">
          <h3>コンテキスト項目（タスク作成時の入力フィールド）</h3>
          {contextFields.map((f, i) => (
            <div className="row" key={i} style={{ alignItems: "flex-end", marginBottom: 8 }}>
              <div className="col">
                <input
                  placeholder="フィールド名（例: route_from）"
                  value={f.name}
                  onChange={(e) =>
                    setContextFields((p) => p.map((x, j) => (j === i ? { ...x, name: e.target.value } : x)))
                  }
                />
              </div>
              <div style={{ width: 160 }}>
                <select
                  value={f.type}
                  onChange={(e) =>
                    setContextFields((p) => p.map((x, j) => (j === i ? { ...x, type: e.target.value } : x)))
                  }
                >
                  <option value="string">string</option>
                  <option value="number">number</option>
                  <option value="boolean">boolean</option>
                </select>
              </div>
              <button type="button" className="btn-ghost" onClick={() => setContextFields((p) => p.filter((_, j) => j !== i))}>
                削除
              </button>
            </div>
          ))}
          <button type="button" onClick={() => setContextFields((p) => [...p, { name: "", type: "string" }])}>
            ＋ 項目を追加
          </button>
        </div>

        <div className="panel">
          <h3>ステップ（頂点）と遷移（辺）</h3>
          <p className="muted" style={{ fontSize: 12, marginTop: -4 }}>
            ガード例: <code>context.amount &gt; 0</code> ·{" "}
            <code>approval_approved("step")</code> · <code>elicit_yes("step")</code> ·{" "}
            <code>trigger("step")</code> · <code>actor.role == "accounting"</code>
          </p>

          {steps.map((s) => (
            <div key={s._id} className="stepcard" style={{ display: "block" }}>
              <div className="row" style={{ alignItems: "flex-end" }}>
                <div className="col">
                  <label>ステップキー</label>
                  <input value={s.key} onChange={(e) => patchStep(s._id, { key: e.target.value })} placeholder="price_check" />
                </div>
                <div className="col">
                  <label>タイトル</label>
                  <input value={s.title} onChange={(e) => patchStep(s._id, { title: e.target.value })} placeholder="運賃を照会" />
                </div>
                <button type="button" className="btn-red" onClick={() => setSteps((p) => p.filter((x) => x._id !== s._id))}>
                  ステップ削除
                </button>
              </div>

              <div className="row">
                <div className="col">
                  <label>種別</label>
                  <select value={s.kind} onChange={(e) => patchStep(s._id, { kind: e.target.value, gate_kind: e.target.value === "human_gate" ? s.gate_kind || "approval" : "" })}>
                    {KINDS.map((k) => <option key={k.v} value={k.v}>{k.l}</option>)}
                  </select>
                </div>
                {s.kind === "human_gate" && (
                  <div className="col">
                    <label>ゲート種別</label>
                    <select value={s.gate_kind} onChange={(e) => patchStep(s._id, { gate_kind: e.target.value })}>
                      {GATE_KINDS.map((g) => <option key={g.v} value={g.v}>{g.l}</option>)}
                    </select>
                  </div>
                )}
                <div className="col">
                  <label>状態ラベル（enters_state）</label>
                  <input value={s.enters_state} onChange={(e) => patchStep(s._id, { enters_state: e.target.value })} placeholder={s.key} />
                </div>
              </div>

              <div className="row" style={{ alignItems: "center" }}>
                {(s.kind === "agent_action" || s.kind === "system_action") && !s.terminal && (
                  <div className="col">
                    <label>実行ツール</label>
                    <select value={s.tool_key} onChange={(e) => patchStep(s._id, { tool_key: e.target.value })}>
                      <option value="">（選択）</option>
                      {(tools.data ?? []).map((t) => (
                        <option key={t.key} value={t.key}>{t.key}（{t.side_effect_class}）</option>
                      ))}
                    </select>
                    {(() => {
                      const t = tools.data?.find((x) => x.key === s.tool_key);
                      return t?.outputs?.length ? (
                        <div className="muted" style={{ fontSize: 11, marginTop: 4 }}>
                          出力: {t.outputs.map((o) => o.name).join(", ")}
                        </div>
                      ) : null;
                    })()}
                  </div>
                )}
                <label style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 20 }}>
                  <input
                    type="checkbox"
                    style={{ width: "auto" }}
                    checked={s.terminal}
                    onChange={(e) => patchStep(s._id, { terminal: e.target.checked, transitions: e.target.checked ? [] : s.transitions })}
                  />
                  終端ステップ
                </label>
              </div>

              {!s.terminal && (
                <div style={{ marginTop: 8 }}>
                  <label>遷移（条件が成立した最初の辺へ進む）</label>
                  {s.transitions.map((t, ti) => (
                    <div className="row" key={ti} style={{ alignItems: "flex-end", marginBottom: 6 }}>
                      <div style={{ width: 200 }}>
                        <select
                          value={t.to}
                          onChange={(e) =>
                            patchStep(s._id, { transitions: s.transitions.map((x, j) => (j === ti ? { ...x, to: e.target.value } : x)) })
                          }
                        >
                          <option value="">— 遷移先 —</option>
                          {stepKeys.filter((k) => k !== s.key).map((k) => <option key={k} value={k}>{k}</option>)}
                        </select>
                      </div>
                      <div className="col">
                        <input
                          placeholder='ガード（空=常に成立）'
                          value={t.guard}
                          onChange={(e) =>
                            patchStep(s._id, { transitions: s.transitions.map((x, j) => (j === ti ? { ...x, guard: e.target.value } : x)) })
                          }
                        />
                      </div>
                      <button type="button" className="btn-ghost" onClick={() => patchStep(s._id, { transitions: s.transitions.filter((_, j) => j !== ti) })}>
                        削除
                      </button>
                    </div>
                  ))}
                  <button type="button" onClick={() => patchStep(s._id, { transitions: [...s.transitions, { to: "", guard: "" }] })}>
                    ＋ 遷移を追加
                  </button>
                </div>
              )}
            </div>
          ))}

          <button type="button" onClick={() => setSteps((p) => [...p, blankStep("", "agent_action")])} style={{ marginTop: 8 }}>
            ＋ ステップを追加
          </button>
        </div>

        <div className="panel">
          <h3>開始ステップ</h3>
          <select value={entryStep} onChange={(e) => setEntryStep(e.target.value)} style={{ maxWidth: 320 }}>
            <option value="">— 選択 —</option>
            {stepKeys.map((k) => <option key={k} value={k}>{k}</option>)}
          </select>
        </div>

        {err && <p className="err">{err}</p>}
        <div className="actions">
          <button className="btn-primary" disabled={busy}>{busy ? "登録中…" : "フローを登録"}</button>
          <button type="button" className="btn-ghost" onClick={() => navigate("/flows")}>キャンセル</button>
        </div>
      </form>
    </div>
  );
}
