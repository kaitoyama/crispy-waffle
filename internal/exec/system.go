package exec

import "github.com/kaitoyama/crispy-waffle/internal/workflow"

// preApplicationSubmit posts the pre-application (irreversible). The returned id
// is derived from the idempotency key so re-execution yields the same value;
// the engine additionally guarantees the side effect runs at most once.
func preApplicationSubmit(in workflow.ExecInput) (map[string]any, error) {
	return map[string]any{
		"pre_approval_id":   "pre-" + shortHash(in.IdempotencyKey),
		"pre_application_submitted": true,
	}, nil
}

// paymentExecute settles the expense (irreversible/monetary). Idempotent by the
// engine's ledger check plus this deterministic, key-derived payment id.
func paymentExecute(in workflow.ExecInput) (map[string]any, error) {
	out := map[string]any{
		"settled":    true,
		"payment_id": "pay-" + shortHash(in.IdempotencyKey),
	}
	if amt, ok := toFloat(in.Context["amount"]); ok {
		out["paid_amount"] = amt
	}
	return out, nil
}
