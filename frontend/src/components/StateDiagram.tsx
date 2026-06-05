import { useEffect, useRef } from "react";
import mermaid from "mermaid";
import type { WorkflowDefinition } from "../api/types";

mermaid.initialize({ startOnLoad: false, theme: "dark", securityLevel: "loose" });

function sanitize(s: string): string {
  return s.replace(/"/g, "'");
}

// Builds a stateDiagram-v2 from the definition and highlights the current step.
function toMermaid(def: WorkflowDefinition, current?: string): string {
  const lines = ["stateDiagram-v2", `  [*] --> ${def.entry_step}`];
  for (const key of Object.keys(def.steps)) {
    const step = def.steps[key];
    lines.push(`  state "${sanitize(step.title || key)}" as ${key}`);
  }
  for (const key of Object.keys(def.steps)) {
    const step = def.steps[key];
    if (step.terminal) {
      lines.push(`  ${key} --> [*]`);
      continue;
    }
    for (const t of step.transitions ?? []) {
      const label = t.guard ? `: ${sanitize(t.guard)}` : "";
      lines.push(`  ${key} --> ${t.to}${label}`);
    }
  }
  lines.push("  classDef current fill:#5aa9ff,color:#06121f,stroke:#5aa9ff,stroke-width:2px;");
  if (current) lines.push(`  class ${current} current`);
  return lines.join("\n");
}

export function StateDiagram({
  def,
  current,
}: {
  def: WorkflowDefinition;
  current?: string;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    let cancelled = false;
    const graph = toMermaid(def, current);
    mermaid
      .render(`sd-${Math.random().toString(36).slice(2)}`, graph)
      .then(({ svg }) => {
        if (!cancelled && ref.current) ref.current.innerHTML = svg;
      })
      .catch((e) => {
        if (ref.current) ref.current.textContent = String(e);
      });
    return () => {
      cancelled = true;
    };
  }, [def, current]);
  return <div className="mermaid-box" ref={ref} />;
}
