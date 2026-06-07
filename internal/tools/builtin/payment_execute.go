package builtin

import (
	"context"

	"github.com/kaitoyama/crispy-waffle/internal/tools"
)

// PaymentExecute settles the expense (irreversible/monetary). Idempotent by the
// engine's ledger check plus this deterministic, key-derived payment id. The
// amount scope dimension is bound by the authorization gate against the actor's
// capability cap (e.g. acc-bot @ {amount_cap: 20000}).
type PaymentExecute struct{}

func (PaymentExecute) Spec() tools.Spec {
	return tools.Spec{
		Key:             "payment.execute",
		DisplayName:     "精算（支払）実行",
		SideEffectClass: tools.IrreversibleMonetary,
		Description:     "精算（送金）を実行する（不可逆・冪等必須）",
		Inputs:          []tools.Field{{Name: "amount", Type: "number", Required: true}},
		Outputs: []tools.Field{
			{Name: "settled", Type: "boolean"},
			{Name: "payment_id", Type: "string"},
			{Name: "paid_amount", Type: "number"},
		},
		ScopeDimensions: []string{"amount"},
	}
}

func (PaymentExecute) Execute(_ context.Context, in tools.Input) (map[string]any, error) {
	out := map[string]any{
		"settled":    true,
		"payment_id": "pay-" + shortHash(in.IdempotencyKey),
	}
	if amt, ok := toFloat(in.Context["amount"]); ok {
		out["paid_amount"] = amt
	}
	return out, nil
}
