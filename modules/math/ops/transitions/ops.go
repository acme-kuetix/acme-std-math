// Package transitions implements the math/ops service — basic arithmetic
// primitives for WSL workflows. All inputs are interface{} because WSL
// JSON-decodes numbers as float64 (or int for whole literals), and a Go
// method declaring int or float64 panics on the wrong concrete type.
// Coercion happens at runtime via toFloat.
//
// PROMOTION-CANDIDATE: stable since Wave 5, no acme-* deps, used in 6 packages.
// Provides Add/Sub/Mul/Div/Round/AssertPositive. No std-* equivalent.
// Consider promoting to std-math after kuetix review.
package transitions

import (
	"fmt"
	"math"
	"strconv"

	"github.com/kuetix/engine/engine/domain"
	"github.com/kuetix/engine/engine/domain/interfaces"
	"github.com/kuetix/engine/engine/workflow"
)

var _ interfaces.ServiceTransitions = (*opsTransitions)(nil)

type opsTransitions struct {
	workflow.BaseServiceTransition
}

func NewOpsTransitions() interfaces.ServiceTransitions {
	return &opsTransitions{}
}

// NewOpsTransitionsConcrete returns the concrete *opsTransitions pointer.
// Composition-layer callers (e.g. ledger.ValidateMoveBalance delegating to
// ops.AssertBalanced) use this to invoke primitive methods directly from Go
// without the interfaces.ServiceTransitions indirection.
func NewOpsTransitionsConcrete() *opsTransitions {
	return &opsTransitions{}
}

// toFloat coerces int*, float*, and numeric strings to float64.
// Non-coercible input returns 0 — callers that need strict validation
// should validate at the WSL layer (the engine already rejects missing
// required args before the transition is called).
func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case bool:
		// Don't coerce booleans — they're almost certainly a bug.
		return 0
	case string:
		if parsed, err := strconv.ParseFloat(n, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// Add returns a + b.
// WSL: math/ops.Add(a: $qty, b: $price)
func (t *opsTransitions) Add(a interface{}, b interface{}) (r domain.FlowStepResult) {
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{
		"result": toFloat(a) + toFloat(b),
	}
	return
}

// Sub returns a - b.
// WSL: math/ops.Sub(a: $base, b: $taxExclusiveBase)
func (t *opsTransitions) Sub(a interface{}, b interface{}) (r domain.FlowStepResult) {
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{
		"result": toFloat(a) - toFloat(b),
	}
	return
}

// Mul returns a * b.
// WSL: math/ops.Mul(a: $qty, b: $price)
func (t *opsTransitions) Mul(a interface{}, b interface{}) (r domain.FlowStepResult) {
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{
		"result": toFloat(a) * toFloat(b),
	}
	return
}

// Div returns a / b. Returns 400 division_by_zero if b == 0.
// WSL: math/ops.Div(a: $total, b: $count)
func (t *opsTransitions) Div(a interface{}, b interface{}) (r domain.FlowStepResult) {
	bv := toFloat(b)
	if bv == 0 {
		r.Success = false
		r.StatusCode = 400
		r.Error = fmt.Errorf("math/ops: division by zero")
		r.Response = map[string]interface{}{
			"code":    "division_by_zero",
			"message": "cannot divide by zero",
		}
		return
	}
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{
		"result": toFloat(a) / bv,
	}
	return
}

// Round rounds value to `decimals` decimal places (half away from zero).
// WSL: math/ops.Round(value: $total, decimals: 2)
func (t *opsTransitions) Round(value interface{}, decimals int) (r domain.FlowStepResult) {
	v := toFloat(value)
	if decimals < 0 {
		decimals = 0
	}
	pow := math.Pow(10, float64(decimals))
	rounded := math.Round(v*pow) / pow
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{
		"result": rounded,
	}
	return
}

// AssertPositive fails with 400 if value <= 0. Returns the value as float64
// on success. Used for validating rates, amounts, factors, etc.
// WSL: math/ops.AssertPositive(value: $json.rate, code: "invalid_rate", message: "rate must be positive")
func (t *opsTransitions) AssertPositive(value interface{}, code string, message string) (r domain.FlowStepResult) {
	v := toFloat(value)
	if v <= 0 {
		if code == "" {
			code = "invalid_value"
		}
		if message == "" {
			message = "value must be positive"
		}
		r.Success = false
		r.StatusCode = 400
		r.Error = fmt.Errorf("%s", message)
		r.Response = map[string]interface{}{"code": code, "message": message}
		return
	}
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{"value": v, "valid": true}
	return
}

// AssertGreaterThan fails with 409 if value <= threshold. Returns the value
// on success. This is the WSL equivalent of `if x > y` — on-success path
// continues when value exceeds threshold; on-error path handles the
// not-greater case. Used to branch on counts (e.g. count > 0 before
// proceeding with a multi-step operation).
// WSL: math/ops.AssertGreaterThan(value: $ListRes.response.count, threshold: 0, code: "no_entries", message: "at least one entry required")
func (t *opsTransitions) AssertGreaterThan(value interface{}, threshold interface{}, code string, message string) (r domain.FlowStepResult) {
	v := toFloat(value)
	t2 := toFloat(threshold)
	if v <= t2 {
		if code == "" {
			code = "not_greater"
		}
		if message == "" {
			message = "value must be greater than threshold"
		}
		r.Success = false
		r.StatusCode = 409
		r.Error = fmt.Errorf("%s", message)
		r.Response = map[string]interface{}{"code": code, "message": message, "value": v, "threshold": t2}
		return
	}
	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{"value": v, "threshold": t2, "valid": true}
	return
}

// toMapSlice coerces an interface{} (typically []interface{} from WSL/JSON)
// into a slice of maps. Used by AssertBalanced to parse the lines slice.
func toMapSlice(v interface{}) []map[string]interface{} {
	switch s := v.(type) {
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(s))
		for _, item := range s {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	case []map[string]interface{}:
		return s
	}
	return nil
}

// AssertBalanced validates that a slice of lines forms a balanced double-entry
// move: at least 2 lines, at least one debit and one credit, sum of debits equals
// sum of credits (within epsilon). Each line is a map with "debit" and "credit"
// numeric fields. Returns {balanced: true, totalDebit, totalCredit} on success,
// 400 error with a descriptive message on failure.
//
// WSL: math/ops/ops.AssertBalanced(lines: $lines)
//
// PROMOTION-CANDIDATE: stable since Wave 29, no acme-* deps, used in acme-ledger.
// Consider promoting to std-math after kuetix review.
func (t *opsTransitions) AssertBalanced(lines interface{}) (r domain.FlowStepResult) {
	parsed := toMapSlice(lines)
	if len(parsed) < 2 {
		r.Success = false
		r.StatusCode = 400
		r.Error = fmt.Errorf("move must have at least 2 lines (has %d)", len(parsed))
		r.Response = map[string]interface{}{
			"code":    "invalid_move",
			"message": fmt.Sprintf("move must have at least 2 lines (has %d)", len(parsed)),
		}
		return
	}

	var totalDebit, totalCredit float64
	hasDebit, hasCredit := false, false
	for _, l := range parsed {
		d := toFloat(l["debit"])
		c := toFloat(l["credit"])
		if d < 0 || c < 0 {
			r.Success = false
			r.StatusCode = 400
			r.Error = fmt.Errorf("debit and credit must be non-negative")
			r.Response = map[string]interface{}{
				"code":    "invalid_move",
				"message": "debit and credit must be non-negative",
			}
			return
		}
		if d == 0 && c == 0 {
			r.Success = false
			r.StatusCode = 400
			r.Error = fmt.Errorf("either debit or credit must be non-zero")
			r.Response = map[string]interface{}{
				"code":    "invalid_move",
				"message": "either debit or credit must be non-zero",
			}
			return
		}
		if d > 0 && c > 0 {
			r.Success = false
			r.StatusCode = 400
			r.Error = fmt.Errorf("a line cannot have both debit and credit")
			r.Response = map[string]interface{}{
				"code":    "invalid_move",
				"message": "a line cannot have both debit and credit",
			}
			return
		}
		totalDebit += d
		totalCredit += c
		if d > 0 {
			hasDebit = true
		}
		if c > 0 {
			hasCredit = true
		}
	}
	if !hasDebit || !hasCredit {
		r.Success = false
		r.StatusCode = 400
		r.Error = fmt.Errorf("move must have at least one debit and one credit line")
		r.Response = map[string]interface{}{
			"code":    "invalid_move",
			"message": "move must have at least one debit and one credit line",
		}
		return
	}
	if math.Abs(totalDebit-totalCredit) > 1e-6 {
		r.Success = false
		r.StatusCode = 400
		r.Error = fmt.Errorf("move is not balanced: debits=%.2f, credits=%.2f", totalDebit, totalCredit)
		r.Response = map[string]interface{}{
			"code":    "unbalanced_move",
			"message": fmt.Sprintf("move is not balanced: debits=%.2f, credits=%.2f", totalDebit, totalCredit),
		}
		return
	}

	r.Success = true
	r.StatusCode = 200
	r.Response = map[string]interface{}{
		"balanced":    true,
		"totalDebit":  totalDebit,
		"totalCredit": totalCredit,
	}
	return
}
