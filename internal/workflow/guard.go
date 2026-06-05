package workflow

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

// Guard evaluation uses a deliberately tiny, safe, non-Turing predicate set
// over the folded RunState (docs/03 §4). This covers the reference workflow
// without the risk of an embedded expression language. Clauses may be joined
// with "&&".
//
// Supported clauses:
//
//	(empty)                       -> always true
//	context.<key> <op> <number>   -> op in > >= < <= == !=
//	approval_approved("step")
//	approval_rejected("step")
//	approval_changes("step")
//	elicit_yes("step")
//	elicit_no("step")
//	trigger("step")
//	actor.role == "role"

var (
	reCmp    = regexp.MustCompile(`^context\.([a-zA-Z_][\w]*)\s*(>=|<=|==|!=|>|<)\s*(-?\d+(?:\.\d+)?)$`)
	reCall   = regexp.MustCompile(`^([a-z_]+)\("([^"]*)"\)$`)
	reRole   = regexp.MustCompile(`^actor\.role\s*==\s*"([^"]*)"$`)
)

// EvalGuard reports whether the guard holds for the given state and actor.
func EvalGuard(guard string, s RunState, actor domain.Actor) bool {
	guard = strings.TrimSpace(guard)
	if guard == "" {
		return true
	}
	for _, clause := range strings.Split(guard, "&&") {
		if !evalClause(strings.TrimSpace(clause), s, actor) {
			return false
		}
	}
	return true
}

func evalClause(c string, s RunState, actor domain.Actor) bool {
	if m := reRole.FindStringSubmatch(c); m != nil {
		return actor.HasRole(m[1])
	}
	if m := reCmp.FindStringSubmatch(c); m != nil {
		left, ok := toFloat(s.Context[m[1]])
		if !ok {
			return false
		}
		right, _ := strconv.ParseFloat(m[3], 64)
		switch m[2] {
		case ">":
			return left > right
		case ">=":
			return left >= right
		case "<":
			return left < right
		case "<=":
			return left <= right
		case "==":
			return left == right
		case "!=":
			return left != right
		}
	}
	if m := reCall.FindStringSubmatch(c); m != nil {
		fn, arg := m[1], m[2]
		switch fn {
		case "approval_approved":
			return s.Approvals[arg] == domain.DecisionApproved
		case "approval_rejected":
			return s.Approvals[arg] == domain.DecisionRejected
		case "approval_changes":
			return s.Approvals[arg] == domain.DecisionChangesRequested
		case "elicit_yes":
			v, ok := s.ElicitAnswers[arg]
			return ok && v
		case "elicit_no":
			v, ok := s.ElicitAnswers[arg]
			return ok && !v
		case "trigger":
			return s.Triggers[arg]
		}
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
