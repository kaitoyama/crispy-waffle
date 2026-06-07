package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// --- request shapes for the flow builder UI ---

type ContextFieldInput struct {
	Name string `json:"name"`
	Type string `json:"type"` // string | number | boolean
}

type TransitionInput struct {
	To    string `json:"to"`
	Guard string `json:"guard"`
}

type StepInput struct {
	Key         string            `json:"key"`
	Title       string            `json:"title"`
	Kind        string            `json:"kind"`
	GateKind    string            `json:"gate_kind"`
	EntersState string            `json:"enters_state"`
	Terminal    bool              `json:"terminal"`
	ToolKey     string            `json:"tool_key"`
	ToolScope   map[string]any    `json:"tool_scope"`
	Transitions []TransitionInput `json:"transitions"`
}

type RegisterFlowRequest struct {
	Key           string              `json:"key"`
	DisplayName   string              `json:"display_name"`
	EntryStep     string              `json:"entry_step"`
	ContextFields []ContextFieldInput `json:"context_fields"`
	Steps         []StepInput         `json:"steps"`
}

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.]*$`)

// RegisterFlow validates a flow definition from the UI, persists it as the next
// version of its key, registers it in the engine registry, and upserts a
// TaskType so tasks of this flow can be created. Existing runs keep their pinned
// version (docs/02 §11-6).
func (s *Service) RegisterFlow(ctx context.Context, req RegisterFlowRequest) (workflow.WorkflowDefinition, error) {
	def, err := s.buildDefinition(ctx, req)
	if err != nil {
		return workflow.WorkflowDefinition{}, err
	}
	if err := s.Store.UpsertDefinition(ctx, def); err != nil {
		return workflow.WorkflowDefinition{}, err
	}
	s.Registry.Register(def)

	// Build a context schema (field -> type hint) and a TaskType pointing here.
	schema := map[string]any{}
	for _, f := range req.ContextFields {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			continue
		}
		t := f.Type
		if t == "" {
			t = "string"
		}
		schema[name] = t
	}
	schemaJSON, _ := json.Marshal(schema)
	if err := s.Store.UpsertTaskType(ctx, domain.TaskType{
		Key:                def.Key,
		DisplayName:        def.DisplayName,
		ContextSchema:      schemaJSON,
		DefaultWorkflowKey: def.Key,
		DefaultWorkflowVer: def.Version,
	}); err != nil {
		return workflow.WorkflowDefinition{}, err
	}
	return def, nil
}

// buildDefinition validates the request and assembles a WorkflowDefinition with
// the next version number for its key.
func (s *Service) buildDefinition(ctx context.Context, req RegisterFlowRequest) (workflow.WorkflowDefinition, error) {
	key := strings.TrimSpace(req.Key)
	if !keyPattern.MatchString(key) {
		return workflow.WorkflowDefinition{}, fmt.Errorf("key must match %s (例: expense.lodging)", keyPattern.String())
	}
	if len(req.Steps) == 0 {
		return workflow.WorkflowDefinition{}, fmt.Errorf("少なくとも1つのステップが必要です")
	}

	steps := map[string]workflow.StepDef{}
	for _, in := range req.Steps {
		sk := strings.TrimSpace(in.Key)
		if !keyPattern.MatchString(sk) {
			return workflow.WorkflowDefinition{}, fmt.Errorf("ステップキー %q が不正です", in.Key)
		}
		if _, dup := steps[sk]; dup {
			return workflow.WorkflowDefinition{}, fmt.Errorf("ステップキー %q が重複しています", sk)
		}
		kind := workflow.StepKind(in.Kind)
		switch kind {
		case workflow.KindAgentAction, workflow.KindSystemAction, workflow.KindHumanGate, workflow.KindSubWorkflow:
		default:
			return workflow.WorkflowDefinition{}, fmt.Errorf("ステップ %q の種別 %q が不正です", sk, in.Kind)
		}

		st := workflow.StepDef{
			Key:         sk,
			Title:       strings.TrimSpace(in.Title),
			Kind:        kind,
			EntersState: domain.TaskStatus(orDefault(in.EntersState, sk)),
			Terminal:    in.Terminal,
		}

		if kind == workflow.KindHumanGate {
			gk := workflow.GateKind(in.GateKind)
			switch gk {
			case workflow.GateApproval, workflow.GateElicitation, workflow.GateTrigger:
				st.GateKind = gk
			default:
				return workflow.WorkflowDefinition{}, fmt.Errorf("人間ゲート %q には gate_kind（approval/elicitation/trigger）が必要です", sk)
			}
		}

		// Automatic, non-terminal steps run exactly one bound tool.
		needsTool := (kind == workflow.KindAgentAction || kind == workflow.KindSystemAction) && !in.Terminal
		if tk := strings.TrimSpace(in.ToolKey); tk != "" {
			if s.Catalog != nil {
				if _, ok := s.Catalog.Get(tk); !ok {
					return workflow.WorkflowDefinition{}, fmt.Errorf("ツール %q はカタログに存在しません", tk)
				}
			}
			st.ToolBindings = []workflow.ToolBinding{{ToolKey: tk, Scope: in.ToolScope}}
		} else if needsTool {
			return workflow.WorkflowDefinition{}, fmt.Errorf("%s ステップ %q にはツールを1つ束縛してください", kind, sk)
		}

		for _, t := range in.Transitions {
			to := strings.TrimSpace(t.To)
			if to == "" {
				continue
			}
			st.Transitions = append(st.Transitions, workflow.Transition{To: to, Guard: strings.TrimSpace(t.Guard)})
		}
		steps[sk] = st
	}

	entry := strings.TrimSpace(req.EntryStep)
	if _, ok := steps[entry]; !ok {
		return workflow.WorkflowDefinition{}, fmt.Errorf("entry_step %q がステップに存在しません", entry)
	}

	// All transition targets must exist; non-terminal non-gate steps must lead
	// somewhere (a dead end would fail at runtime).
	for _, st := range steps {
		for _, t := range st.Transitions {
			if _, ok := steps[t.To]; !ok {
				return workflow.WorkflowDefinition{}, fmt.Errorf("ステップ %q の遷移先 %q が存在しません", st.Key, t.To)
			}
		}
		if !st.Terminal && len(st.Transitions) == 0 {
			return workflow.WorkflowDefinition{}, fmt.Errorf("非終端ステップ %q には少なくとも1つの遷移が必要です", st.Key)
		}
	}

	ver := 1
	if max, ok, err := s.Store.MaxVersion(ctx, key); err != nil {
		return workflow.WorkflowDefinition{}, err
	} else if ok {
		ver = max + 1
	}

	return workflow.WorkflowDefinition{
		Key:         key,
		Version:     ver,
		DisplayName: orDefault(strings.TrimSpace(req.DisplayName), key),
		EntryStep:   entry,
		Steps:       steps,
	}, nil
}

// ListDefinitions returns all registered workflow definitions.
func (s *Service) ListDefinitions(ctx context.Context) ([]workflow.WorkflowDefinition, error) {
	return s.Store.ListDefinitions(ctx)
}

// ListTaskTypes returns all task types (for the New Task picker).
func (s *Service) ListTaskTypes(ctx context.Context) ([]domain.TaskType, error) {
	return s.Store.ListTaskTypes(ctx)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
