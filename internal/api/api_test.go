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
	eng := &workflow.Engine{Events: st, Runs: st, Tasks: st, Actors: st, Registry: reg, Executor: exec.NewExecutor()}
	svc := service.New(st, eng, reg, seed.ActorAccBot)
	if err := seed.Install(context.Background(), st, reg, svc); err != nil {
		t.Fatal(err)
	}
	srv := &api.Server{Svc: svc, Catalog: tools.DefaultCatalog(), Registry: reg}
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
