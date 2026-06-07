package builtin

import (
	"context"
	"fmt"

	"github.com/kaitoyama/crispy-waffle/internal/tools"
)

// NotifySend is an example of adding a tool purely in code: a reversible_write
// "send a notification" capability. It shows the whole extension surface — a
// Spec (which feeds the catalog API and the flow-builder picker) and an Execute.
// Swap the body for a real channel (traQ, mail, webhook) without touching wiring.
type NotifySend struct{}

func (NotifySend) Spec() tools.Spec {
	return tools.Spec{
		Key:             "notify.send",
		DisplayName:     "通知の送信",
		SideEffectClass: tools.ReversibleWrite,
		Description:     "担当者へ通知を送る（例示用の組み込みツール）",
		Inputs: []tools.Field{
			{Name: "message", Type: "string", Required: true},
		},
		Outputs: []tools.Field{
			{Name: "notified", Type: "boolean"},
			{Name: "notify_channel", Type: "string"},
			{Name: "notify_id", Type: "string"},
		},
	}
}

func (NotifySend) Execute(_ context.Context, in tools.Input) (map[string]any, error) {
	msg := ctxString(in.Context, "message")
	if msg == "" {
		return nil, fmt.Errorf("notify.send: message required")
	}
	return map[string]any{
		"notified":       true,
		"notify_channel": "log",
		"notify_id":      "ntf-" + shortHash(in.IdempotencyKey),
	}, nil
}
