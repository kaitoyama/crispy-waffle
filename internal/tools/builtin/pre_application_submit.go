package builtin

import (
	"context"

	"github.com/kaitoyama/crispy-waffle/internal/tools"
)

// PreApplicationSubmit posts the pre-application (irreversible). The returned id
// is derived from the idempotency key so re-execution yields the same value; the
// engine additionally guarantees the effect runs at most once.
type PreApplicationSubmit struct{}

func (PreApplicationSubmit) Spec() tools.Spec {
	return tools.Spec{
		Key:             "pre_application.submit",
		DisplayName:     "事前申請の提出",
		SideEffectClass: tools.IrreversibleMonetary,
		Description:     "事前申請を投稿する（不可逆）",
		Outputs: []tools.Field{
			{Name: "pre_approval_id", Type: "string"},
			{Name: "pre_application_submitted", Type: "boolean"},
		},
	}
}

func (PreApplicationSubmit) Execute(_ context.Context, in tools.Input) (map[string]any, error) {
	return map[string]any{
		"pre_approval_id":           "pre-" + shortHash(in.IdempotencyKey),
		"pre_application_submitted": true,
	}, nil
}
