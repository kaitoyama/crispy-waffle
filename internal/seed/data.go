// Package seed defines the built-in actors, task types and the reference
// expense.transport workflow, and installs them idempotently at startup.
package seed

import (
	"encoding/json"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// Built-in actor ids (stub auth: the frontend picks one via X-Actor-Id).
const (
	ActorTanaka     = domain.ActorID("tanaka")
	ActorAccBot     = domain.ActorID("acc-bot")
	ActorAccountant = domain.ActorID("accountant")
)

// Task statuses for the expense.transport state machine (docs/08 §1.2).
const (
	StDrafting           = domain.TaskStatus("Drafting")
	StConfirmPre         = domain.TaskStatus("ConfirmPre")
	StSubmitting         = domain.TaskStatus("Submitting")
	StAwaitingApproval   = domain.TaskStatus("AwaitingApproval")
	StAwaitingSettlement = domain.TaskStatus("AwaitingSettlement")
	StSettling           = domain.TaskStatus("Settling")
	StSettled            = domain.TaskStatus("Settled")
	StRejected           = domain.TaskStatus("Rejected")
)

// Actors returns the seeded actors. acc-bot is delegated from tanaka and holds
// an attenuated payment capability (amount_cap), per docs/05.
func Actors() []domain.Actor {
	tanaka := ActorTanaka
	return []domain.Actor{
		{
			ID: ActorTanaka, DisplayName: "田中", Kind: domain.ActorHuman,
			IdentityRef: "oidc:tanaka", Roles: []string{"requester"},
		},
		{
			ID: ActorAccBot, DisplayName: "acc-bot", Kind: domain.ActorAgent,
			IdentityRef: "nhi:acc-bot", Roles: []string{"agent"}, DelegatedFrom: &tanaka,
			Capabilities: []domain.Capability{
				{ToolKey: "fare.lookup"},
				{ToolKey: "pre_application.submit"},
				{ToolKey: "payment.execute", Scope: map[string]any{"amount_cap": 20000.0}},
			},
		},
		{
			ID: ActorAccountant, DisplayName: "会計担当", Kind: domain.ActorHuman,
			IdentityRef: "oidc:accountant", Roles: []string{"accounting"},
		},
	}
}

// TaskTypeExpenseTransport is the business template for transport reimbursement.
func TaskTypeExpenseTransport() domain.TaskType {
	schema, _ := json.Marshal(map[string]any{
		"route_from":       "string",
		"route_to":         "string",
		"amount":           "number",
		"currency":         "string",
		"receipt_required": "boolean",
		"pre_approval_id":  "string",
		"settled":          "boolean",
	})
	return domain.TaskType{
		Key:                "expense.transport",
		DisplayName:        "交通費の精算",
		ContextSchema:      schema,
		DefaultWorkflowKey: "expense.transport",
		DefaultWorkflowVer: 1,
	}
}

// ExpenseTransportDefinition is the version-1 workflow (docs/08 §1.2). Steps are
// vertices; transitions are guarded edges. Automatic steps (agent/system) run to
// the next gate; human gates park the run until a signal arrives.
func ExpenseTransportDefinition() workflow.WorkflowDefinition {
	return workflow.WorkflowDefinition{
		Key: "expense.transport", Version: 1, DisplayName: "交通費の精算",
		EntryStep: "price_check",
		Steps: map[string]workflow.StepDef{
			"price_check": {
				Key: "price_check", Kind: workflow.KindAgentAction, Title: "運賃を照会",
				EntersState:  StDrafting,
				ToolBindings: []workflow.ToolBinding{{ToolKey: "fare.lookup"}},
				Transitions:  []workflow.Transition{{To: "confirm_pre", Guard: "context.amount > 0"}},
			},
			"confirm_pre": {
				Key: "confirm_pre", Kind: workflow.KindHumanGate, GateKind: workflow.GateElicitation,
				Title: "この内容で事前申請するか確認", EntersState: StConfirmPre,
				Transitions: []workflow.Transition{
					{To: "submit_pre", Guard: `elicit_yes("confirm_pre")`},
					{To: "price_check", Guard: `elicit_no("confirm_pre")`},
				},
			},
			"submit_pre": {
				Key: "submit_pre", Kind: workflow.KindAgentAction, Title: "事前申請を提出",
				EntersState:  StSubmitting,
				ToolBindings: []workflow.ToolBinding{{ToolKey: "pre_application.submit"}},
				Transitions:  []workflow.Transition{{To: "await_approval"}},
			},
			"await_approval": {
				Key: "await_approval", Kind: workflow.KindHumanGate, GateKind: workflow.GateApproval,
				Title: "会計担当の承認", RequiredRole: "accounting", EntersState: StAwaitingApproval,
				Transitions: []workflow.Transition{
					{To: "awaiting_settlement", Guard: `approval_approved("await_approval")`},
					{To: "rejected", Guard: `approval_rejected("await_approval")`},
				},
			},
			"awaiting_settlement": {
				Key: "awaiting_settlement", Kind: workflow.KindHumanGate, GateKind: workflow.GateTrigger,
				Title: "精算実行を待機", EntersState: StAwaitingSettlement,
				Transitions: []workflow.Transition{
					{To: "settle", Guard: `trigger("awaiting_settlement")`},
				},
			},
			"settle": {
				Key: "settle", Kind: workflow.KindSystemAction, Title: "精算（支払）を実行",
				EntersState: StSettling,
				ToolBindings: []workflow.ToolBinding{
					{ToolKey: "payment.execute", Scope: map[string]any{"amount_cap": 20000.0}},
				},
				Transitions: []workflow.Transition{{To: "done"}},
			},
			"done": {
				Key: "done", Kind: workflow.KindSystemAction, Title: "完了",
				EntersState: StSettled, Terminal: true,
			},
			"rejected": {
				Key: "rejected", Kind: workflow.KindSystemAction, Title: "却下",
				EntersState: StRejected, Terminal: true,
			},
		},
	}
}
