package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kaitoyama/crispy-waffle/internal/api"
	"github.com/kaitoyama/crispy-waffle/internal/exec"
	"github.com/kaitoyama/crispy-waffle/internal/seed"
	"github.com/kaitoyama/crispy-waffle/internal/service"
	"github.com/kaitoyama/crispy-waffle/internal/store"
	"github.com/kaitoyama/crispy-waffle/internal/tools"
	"github.com/kaitoyama/crispy-waffle/internal/tools/builtin"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	reg := workflow.NewRegistry()
	tr := tools.NewRegistry()
	builtin.Register(tr)
	eng := &workflow.Engine{Events: st, Runs: st, Tasks: st, Actors: st, Registry: reg, Executor: exec.NewExecutor(tr)}
	svc := service.New(st, eng, reg, seed.ActorAccBot)
	svc.Tools = tr
	if err := seed.Install(context.Background(), st, reg, svc); err != nil {
		t.Fatal(err)
	}
	srv := &api.Server{Svc: svc, Tools: tr, Registry: reg}
	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Handler())
	return httptest.NewServer(mux)
}

func do(t *testing.T, ts *httptest.Server, method, path, actor string, body any) map[string]any {
	t.Helper()
	var rdr *bytes.Buffer = bytes.NewBuffer(nil)
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewBuffer(b)
	}
	req, _ := http.NewRequest(method, ts.URL+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if actor != "" {
		req.Header.Set("X-Actor-Id", actor)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode >= 400 {
		t.Fatalf("%s %s -> %d: %v", method, path, resp.StatusCode, out)
	}
	return out
}

func TestAPI_RegisterFlowAndRun(t *testing.T) {
	ts := newServer(t)
	defer ts.Close()

	// Register a tool-free approval flow via the API (as the flow builder does).
	def := do(t, ts, "POST", "/api/workflow-definitions", "tanaka", map[string]any{
		"key":            "request.simple",
		"display_name":   "簡易承認",
		"entry_step":     "ask",
		"context_fields": []map[string]any{{"name": "subject", "type": "string"}},
		"steps": []map[string]any{
			{"key": "ask", "title": "承認待ち", "kind": "human_gate", "gate_kind": "approval", "enters_state": "Pending",
				"transitions": []map[string]any{
					{"to": "done", "guard": `approval_approved("ask")`},
					{"to": "rejected", "guard": `approval_rejected("ask")`},
				}},
			{"key": "done", "title": "承認済み", "kind": "system_action", "enters_state": "Approved", "terminal": true},
			{"key": "rejected", "title": "却下", "kind": "system_action", "enters_state": "Rejected", "terminal": true},
		},
	})
	if def["key"] != "request.simple" {
		t.Fatalf("register returned %v", def)
	}

	// Create a task of the new flow and approve it through to completion.
	created := do(t, ts, "POST", "/api/tasks", "tanaka", map[string]any{
		"type":    "request.simple",
		"title":   "備品購入の承認",
		"context": map[string]any{"subject": "モニター"},
	})
	id := created["task"].(map[string]any)["id"].(string)
	if got := created["task"].(map[string]any)["status"]; got != "Pending" {
		t.Fatalf("new flow task status=%v want Pending", got)
	}
	d := do(t, ts, "POST", "/api/approvals/"+id+"/decision", "accountant", map[string]any{"decision": "approved"})
	if got := d["task"].(map[string]any)["status"]; got != "Approved" {
		t.Fatalf("after approve status=%v want Approved", got)
	}
	if got := d["run"].(map[string]any)["status"]; got != "completed" {
		t.Fatalf("run status=%v want completed", got)
	}
}

func TestAPI_CodeToolFlowsEndToEnd(t *testing.T) {
	ts := newServer(t)
	defer ts.Close()

	// The code-registered notify.send tool appears in the catalog automatically.
	var tools []map[string]any
	{
		resp, _ := http.Get(ts.URL + "/api/tools")
		_ = json.NewDecoder(resp.Body).Decode(&tools)
		resp.Body.Close()
	}
	found := false
	for _, tl := range tools {
		if tl["key"] == "notify.send" {
			found = true
		}
	}
	if !found {
		t.Fatalf("notify.send not in catalog: %v", tools)
	}

	// Build a flow that binds the code tool on an automatic step.
	do(t, ts, "POST", "/api/workflow-definitions", "tanaka", map[string]any{
		"key":            "notice.simple",
		"display_name":   "通知フロー",
		"entry_step":     "notify",
		"context_fields": []map[string]any{{"name": "message", "type": "string"}},
		"steps": []map[string]any{
			{"key": "notify", "title": "通知を送る", "kind": "agent_action", "enters_state": "Notifying",
				"tool_key": "notify.send", "transitions": []map[string]any{{"to": "done", "guard": ""}}},
			{"key": "done", "title": "完了", "kind": "system_action", "enters_state": "Done", "terminal": true},
		},
	})

	// Creating a task runs the automatic step (acc-bot is auto-granted the tool)
	// straight through to completion.
	created := do(t, ts, "POST", "/api/tasks", "tanaka", map[string]any{
		"type":    "notice.simple",
		"title":   "お知らせ",
		"context": map[string]any{"message": "こんにちは"},
	})
	task := created["task"].(map[string]any)
	if task["status"] != "Done" {
		t.Fatalf("status=%v want Done", task["status"])
	}
	if created["run"].(map[string]any)["status"] != "completed" {
		t.Fatalf("run=%v want completed", created["run"])
	}
	// The tool invocation is on the ledger.
	invoked := false
	for _, e := range created["events"].([]any) {
		ev := e.(map[string]any)
		if ev["type"] == "tool.invoked" {
			if p, ok := ev["payload"].(map[string]any); ok && p["tool"] == "notify.send" {
				invoked = true
			}
		}
	}
	if !invoked {
		t.Fatalf("notify.send not recorded in ledger")
	}
}

func TestAPI_FullFlow(t *testing.T) {
	ts := newServer(t)
	defer ts.Close()

	// Create a fresh task as tanaka.
	created := do(t, ts, "POST", "/api/tasks", "tanaka", map[string]any{
		"type":    "expense.transport",
		"title":   "テスト精算",
		"context": map[string]any{"route_from": "A", "route_to": "B"},
	})
	task := created["task"].(map[string]any)
	id := task["id"].(string)
	if got := task["status"]; got != "ConfirmPre" {
		t.Fatalf("after create, status=%v want ConfirmPre", got)
	}

	// Elicit yes -> awaiting approval.
	d := do(t, ts, "POST", "/api/tasks/"+id+"/elicitation", "tanaka", map[string]any{"answer": "yes"})
	if got := d["task"].(map[string]any)["status"]; got != "AwaitingApproval" {
		t.Fatalf("after elicit, status=%v want AwaitingApproval", got)
	}

	// Approval inbox should contain this task.
	var inbox []map[string]any
	{
		resp, _ := http.Get(ts.URL + "/api/approvals")
		_ = json.NewDecoder(resp.Body).Decode(&inbox)
		resp.Body.Close()
	}
	found := false
	for _, a := range inbox {
		if a["task_id"] == id {
			found = true
		}
	}
	if !found {
		t.Fatalf("task %s not in approvals inbox %v", id, inbox)
	}

	// Accountant approves -> awaiting settlement.
	d = do(t, ts, "POST", "/api/approvals/"+id+"/decision", "accountant", map[string]any{"decision": "approved"})
	if got := d["task"].(map[string]any)["status"]; got != "AwaitingSettlement" {
		t.Fatalf("after approve, status=%v want AwaitingSettlement", got)
	}

	// Settle -> settled/completed.
	d = do(t, ts, "POST", "/api/tasks/"+id+"/settle", "tanaka", nil)
	if got := d["task"].(map[string]any)["status"]; got != "Settled" {
		t.Fatalf("after settle, status=%v want Settled", got)
	}
	if got := d["run"].(map[string]any)["status"]; got != "completed" {
		t.Fatalf("run status=%v want completed", got)
	}
}
