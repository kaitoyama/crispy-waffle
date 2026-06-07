import type {
  Actor,
  ApprovalRequest,
  RegisterFlowRequest,
  TaskDetail,
  TaskSummary,
  TaskType,
  Tool,
  WorkflowDefinition,
} from "./types";

// The acting actor is a stub-auth header read from localStorage and set by the
// ActorSwitcher. Every request carries it as X-Actor-Id.
export function currentActorId(): string {
  return localStorage.getItem("actorId") || "tanaka";
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: {
      "Content-Type": "application/json",
      "X-Actor-Id": currentActorId(),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    let msg = res.statusText;
    try {
      const j = await res.json();
      if (j.error) msg = j.error;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  return res.json() as Promise<T>;
}

export const api = {
  listTasks: () => req<TaskSummary[]>("GET", "/api/tasks"),
  getTask: (id: string) => req<TaskDetail>("GET", `/api/tasks/${id}`),
  createTask: (body: {
    type: string;
    title: string;
    intent?: string;
    context: Record<string, unknown>;
  }) => req<TaskDetail>("POST", "/api/tasks", body),
  advance: (id: string) => req<TaskDetail>("POST", `/api/tasks/${id}/advance`),
  elicit: (id: string, answer: "yes" | "no") =>
    req<TaskDetail>("POST", `/api/tasks/${id}/elicitation`, { answer }),
  settle: (id: string) => req<TaskDetail>("POST", `/api/tasks/${id}/settle`),
  createLink: (id: string, target_id: string, link_type: string) =>
    req<unknown>("POST", `/api/tasks/${id}/links`, { target_id, link_type }),

  listApprovals: () => req<ApprovalRequest[]>("GET", "/api/approvals"),
  decide: (
    taskId: string,
    decision: string,
    rationale: string,
  ) =>
    req<TaskDetail>("POST", `/api/approvals/${taskId}/decision`, {
      decision,
      rationale,
    }),

  listTools: () => req<Tool[]>("GET", "/api/tools"),
  listActors: () => req<Actor[]>("GET", "/api/actors"),
  getDefinition: (key: string) =>
    req<WorkflowDefinition>("GET", `/api/workflow-definitions/${key}`),

  listTaskTypes: () => req<TaskType[]>("GET", "/api/task-types"),
  listDefinitions: () =>
    req<WorkflowDefinition[]>("GET", "/api/workflow-definitions"),
  registerFlow: (body: RegisterFlowRequest) =>
    req<WorkflowDefinition>("POST", "/api/workflow-definitions", body),
};
