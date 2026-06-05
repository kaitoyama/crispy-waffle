import { useNavigate } from "react-router-dom";
import {
  Background,
  Controls,
  ReactFlow,
  type Edge,
  type Node,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { TaskDetail } from "../api/types";

// Renders the typed task-link graph (docs/02 §4). Nodes are tasks, edges are
// labeled by link_type. Clicking a node navigates to that task.
export function TaskGraph({ detail }: { detail: TaskDetail }) {
  const navigate = useNavigate();
  const focusId = detail.task.id;

  const nodes: Node[] = detail.nodes.map((n, i) => ({
    id: n.id,
    position: { x: (i % 3) * 240, y: Math.floor(i / 3) * 120 },
    data: { label: `${n.title}\n[${n.status}]` },
    style: {
      background: n.id === focusId ? "#222b38" : "#1a212b",
      border: `1px solid ${n.id === focusId ? "#5aa9ff" : "#2c3744"}`,
      color: "#e6edf3",
      borderRadius: 8,
      fontSize: 12,
      width: 200,
      whiteSpace: "pre-line",
    },
  }));

  const edges: Edge[] = detail.links.map((l) => ({
    id: l.id,
    source: l.source_id,
    target: l.target_id,
    label: l.link_type,
    animated: l.link_type === "blocked_by" || l.link_type === "blocks",
    style: { stroke: "#5aa9ff" },
    labelStyle: { fill: "#8b98a8", fontSize: 11 },
  }));

  if (detail.nodes.length <= 1 && detail.links.length === 0) {
    return (
      <p className="muted">
        このタスクにはまだリンクがありません。下のフォームから関連タスクを結べます。
      </p>
    );
  }

  return (
    <div className="graph-wrap">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        fitView
        onNodeClick={(_, node) => navigate(`/tasks/${node.id}`)}
        nodesDraggable
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#2c3744" gap={20} />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}
