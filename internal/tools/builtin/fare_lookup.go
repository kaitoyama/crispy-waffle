package builtin

import (
	"context"
	"fmt"

	"github.com/kaitoyama/crispy-waffle/internal/tools"
)

// FareLookup is a deterministic, offline stand-in for a transit fare API
// (read_only). A real implementation would call an external pricing service
// from Execute — the Spec and wiring stay identical.
type FareLookup struct{}

func (FareLookup) Spec() tools.Spec {
	return tools.Spec{
		Key:             "fare.lookup",
		DisplayName:     "運賃照会",
		SideEffectClass: tools.ReadOnly,
		Description:     "経路から運賃を算出する（参照のみ）",
		Inputs: []tools.Field{
			{Name: "route_from", Type: "string", Required: true},
			{Name: "route_to", Type: "string", Required: true},
		},
		Outputs: []tools.Field{
			{Name: "amount", Type: "number"},
			{Name: "currency", Type: "string"},
			{Name: "fare_basis", Type: "string"},
		},
	}
}

func (FareLookup) Execute(_ context.Context, in tools.Input) (map[string]any, error) {
	from := ctxString(in.Context, "route_from")
	to := ctxString(in.Context, "route_to")
	if from == "" || to == "" {
		return nil, fmt.Errorf("fare.lookup: route_from/route_to required")
	}
	// Stable pseudo-fare in the ¥150–¥2000 range, rounded to ¥10.
	var sum int
	for _, r := range from + "→" + to {
		sum += int(r)
	}
	fare := 150 + (sum*7)%1850
	fare = (fare / 10) * 10
	return map[string]any{
		"amount":     float64(fare),
		"currency":   "JPY",
		"fare_basis": fmt.Sprintf("%s → %s（運賃照会）", from, to),
	}, nil
}
