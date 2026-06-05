package exec

import (
	"fmt"

	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

// fareLookup is a deterministic, offline stand-in for a real fare API
// (read_only). It derives a plausible, stable fare from the route so the demo
// runs with no external dependency. A real implementation would call a transit
// pricing service behind the same toolFunc signature.
func fareLookup(in workflow.ExecInput) (map[string]any, error) {
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
		"amount":   float64(fare),
		"currency": "JPY",
		"fare_basis": fmt.Sprintf("%s → %s（運賃照会）", from, to),
	}, nil
}
