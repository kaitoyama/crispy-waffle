package seed

import (
	"context"
	"log"

	"github.com/kaitoyama/crispy-waffle/internal/service"
	"github.com/kaitoyama/crispy-waffle/internal/store"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

const seededMarker = "seeded.v1"

// Install ensures the reference workflow exists in the DB, loads ALL persisted
// definitions (seed + UI-registered) into the registry, upserts the actors and
// task type (idempotent), and—on first run only—creates a demo task so the
// dashboard isn't empty. Safe to call on every startup.
func Install(ctx context.Context, st *store.Store, reg *workflow.Registry, svc *service.Service) error {
	// Persist the reference flow once, then hydrate the registry from the DB so
	// flows registered through the UI are restored across restarts.
	if _, err := st.GetDefinition(ctx, "expense.transport", 1); err != nil {
		if err := st.UpsertDefinition(ctx, ExpenseTransportDefinition()); err != nil {
			return err
		}
	}
	defs, err := st.ListDefinitions(ctx)
	if err != nil {
		return err
	}
	for _, d := range defs {
		reg.Register(d)
	}

	for _, a := range Actors() {
		if err := st.UpsertActor(ctx, a); err != nil {
			return err
		}
	}
	if err := st.UpsertTaskType(ctx, TaskTypeExpenseTransport()); err != nil {
		return err
	}

	if _, ok, err := st.GetMeta(ctx, seededMarker); err != nil {
		return err
	} else if ok {
		log.Printf("seed: actors/task-type ensured (demo task already seeded)")
		return nil
	}

	// First boot: create a demo task driven to its first wait (confirm_pre).
	_, err = svc.CreateTask(ctx, ActorTanaka, service.CreateTaskRequest{
		Type:   "expense.transport",
		Title:  "6月分 交通費の精算（北千住→御茶ノ水）",
		Intent: "6月の交通費を精算したい",
		Context: map[string]any{
			"route_from": "北千住",
			"route_to":   "御茶ノ水",
		},
	})
	if err != nil {
		return err
	}
	log.Printf("seed: created demo task and seeded expense.transport@1")
	return st.SetMeta(ctx, seededMarker, "1")
}
