import { useQueryClient } from "@tanstack/react-query";
import { useApprovals } from "../hooks/queries";
import { api } from "../api/client";
import { ApprovalCard } from "../components/ApprovalCard";
import { useActor } from "../state/actor";

export function ApprovalsInboxPage() {
  const approvals = useApprovals();
  const qc = useQueryClient();
  const { actorId } = useActor();

  const decide = async (taskId: string, decision: string, rationale: string) => {
    await api.decide(taskId, decision, rationale);
    qc.invalidateQueries({ queryKey: ["approvals"] });
    qc.invalidateQueries({ queryKey: ["tasks"] });
    qc.invalidateQueries({ queryKey: ["task", taskId] });
  };

  return (
    <div>
      <h1 className="page-title">承認インボックス</h1>
      <p className="muted" style={{ marginTop: -8 }}>
        現在の担当者: <strong>{actorId}</strong>（会計担当として承認するには右上で「会計担当」に切替）
      </p>
      {approvals.isLoading && <p className="muted">読み込み中…</p>}
      {approvals.data && approvals.data.length === 0 && (
        <div className="panel"><div className="empty">承認待ちの依頼はありません。</div></div>
      )}
      {approvals.data?.map((r) => (
        <ApprovalCard
          key={r.task_id}
          req={r}
          onDecide={(d, rationale) => decide(r.task_id, d, rationale)}
        />
      ))}
    </div>
  );
}
