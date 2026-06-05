import { useState } from "react";
import { Link } from "react-router-dom";
import type { ApprovalRequest } from "../api/types";

// The minimal-context approval surface (docs/07 §3.1): the approver sees only
// what they need to decide — type, requester, route, amount — not a conversation.
export function ApprovalCard({
  req,
  onDecide,
}: {
  req: ApprovalRequest;
  onDecide: (decision: string, rationale: string) => Promise<void>;
}) {
  const [rationale, setRationale] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const decide = async (decision: string) => {
    setBusy(decision);
    setErr(null);
    try {
      await onDecide(decision, rationale);
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(null);
    }
  };

  const c = req.context || {};
  return (
    <div className="approval-card">
      <div className="ac-head">
        <strong>承認依頼</strong>
        <Link to={`/tasks/${req.task_id}`} className="muted">タスクを開く →</Link>
      </div>
      <dl>
        <dt>種別</dt><dd><span className="kv">{req.type}</span></dd>
        <dt>題名</dt><dd>{req.title}</dd>
        <dt>申請者</dt><dd>{req.requester_id}</dd>
        {c.route_from && <><dt>区間</dt><dd>{c.route_from} → {c.route_to}</dd></>}
        {c.amount != null && <><dt>金額</dt><dd>¥{c.amount}（運賃照会の根拠つき）</dd></>}
        <dt>段階</dt><dd className="kv">{req.step_key}</dd>
      </dl>
      <input
        placeholder="コメント（任意）"
        value={rationale}
        onChange={(e) => setRationale(e.target.value)}
        style={{ marginBottom: 10 }}
      />
      <div className="actions">
        <button className="btn-green" disabled={!!busy} onClick={() => decide("approved")}>
          承認
        </button>
        <button className="btn-red" disabled={!!busy} onClick={() => decide("rejected")}>
          却下
        </button>
        <button disabled={!!busy} onClick={() => decide("changes_requested")}>
          要修正
        </button>
      </div>
      {err && <p className="err" style={{ marginBottom: 0 }}>{err}</p>}
    </div>
  );
}
