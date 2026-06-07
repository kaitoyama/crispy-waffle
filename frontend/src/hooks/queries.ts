import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/client";

export const useTasks = () =>
  useQuery({ queryKey: ["tasks"], queryFn: api.listTasks });

export const useTask = (id: string) =>
  useQuery({ queryKey: ["task", id], queryFn: () => api.getTask(id), enabled: !!id });

export const useApprovals = () =>
  useQuery({ queryKey: ["approvals"], queryFn: api.listApprovals, refetchInterval: 4000 });

export const useTools = () =>
  useQuery({ queryKey: ["tools"], queryFn: api.listTools });

export const useDefinitions = () =>
  useQuery({ queryKey: ["definitions"], queryFn: api.listDefinitions });

export const useTaskTypes = () =>
  useQuery({ queryKey: ["taskTypes"], queryFn: api.listTaskTypes });

export const useActors = () =>
  useQuery({ queryKey: ["actors"], queryFn: api.listActors });

// useTaskAction wraps the workflow-driving mutations and invalidates the task,
// list and inbox so every panel refreshes after an action.
export function useTaskActions(taskId: string) {
  const qc = useQueryClient();
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["task", taskId] });
    qc.invalidateQueries({ queryKey: ["tasks"] });
    qc.invalidateQueries({ queryKey: ["approvals"] });
  };
  return {
    advance: useMutation({ mutationFn: () => api.advance(taskId), onSuccess: invalidate }),
    elicit: useMutation({
      mutationFn: (answer: "yes" | "no") => api.elicit(taskId, answer),
      onSuccess: invalidate,
    }),
    settle: useMutation({ mutationFn: () => api.settle(taskId), onSuccess: invalidate }),
    decide: useMutation({
      mutationFn: (v: { decision: string; rationale: string }) =>
        api.decide(taskId, v.decision, v.rationale),
      onSuccess: invalidate,
    }),
  };
}
