package transitions

import (
	"math"
	"testing"
)

func floatEq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestAdd(t *testing.T) {
	cases := []struct {
		a, b interface{}
		want float64
	}{
		{2, 3, 5},
		{2.5, 3.5, 6.0},
		{"1.1", "2.2", 3.3},
		{int64(10), 5, 15},
		{-1, 1, 0},
	}
	for _, c := range cases {
		r := (&opsTransitions{}).Add(c.a, c.b)
		if !r.Success {
			t.Errorf("Add(%v,%v): unexpected failure: %+v", c.a, c.b, r)
			continue
		}
		got := r.Response.(map[string]interface{})["result"].(float64)
		if !floatEq(got, c.want) {
			t.Errorf("Add(%v,%v): got %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestMul(t *testing.T) {
	r := (&opsTransitions{}).Mul(6, 7)
	if !floatEq(r.Response.(map[string]interface{})["result"].(float64), 42) {
		t.Errorf("Mul(6,7): %+v", r)
	}
	r = (&opsTransitions{}).Mul(2.5, 4)
	if !floatEq(r.Response.(map[string]interface{})["result"].(float64), 10) {
		t.Errorf("Mul(2.5,4): %+v", r)
	}
}

func TestDiv(t *testing.T) {
	r := (&opsTransitions{}).Div(10, 4)
	if !floatEq(r.Response.(map[string]interface{})["result"].(float64), 2.5) {
		t.Errorf("Div(10,4): %+v", r)
	}
}

func TestDiv_ByZero(t *testing.T) {
	r := (&opsTransitions{}).Div(10, 0)
	if r.Success {
		t.Fatal("expected failure")
	}
	if r.StatusCode != 400 {
		t.Errorf("status: got %d, want 400", r.StatusCode)
	}
	resp := r.Response.(map[string]interface{})
	if resp["code"] != "division_by_zero" {
		t.Errorf("code: got %v", resp["code"])
	}
}

func TestRound(t *testing.T) {
	cases := []struct {
		v        interface{}
		decimals int
		want     float64
	}{
		{3.14159, 2, 3.14},
		{3.14159, 4, 3.1416},
		{3.5, 0, 4},
		{2.5, 0, 3}, // half away from zero
		{-2.5, 0, -3},
		{1234.5678, 0, 1235},
		// Note: 1.005 cannot be represented exactly in float64 — it's
		// stored as 1.00499999..., so math.Round(1.005*100)/100 = 1.0.
		// This is the canonical reason acme-std-money exists: float64
		// arithmetic is unsafe for monetary computation.
		{1.005, 2, 1.0},
	}
	for _, c := range cases {
		r := (&opsTransitions{}).Round(c.v, c.decimals)
		if !r.Success {
			t.Errorf("Round(%v,%d): failed: %+v", c.v, c.decimals, r)
			continue
		}
		got := r.Response.(map[string]interface{})["result"].(float64)
		if !floatEq(got, c.want) {
			t.Errorf("Round(%v,%d): got %v, want %v", c.v, c.decimals, got, c.want)
		}
	}
}

func TestRound_NegativeDecimalsClamped(t *testing.T) {
	r := (&opsTransitions{}).Round(123.456, -5)
	if !r.Success {
		t.Fatalf("failed: %+v", r)
	}
	got := r.Response.(map[string]interface{})["result"].(float64)
	if !floatEq(got, 123) {
		t.Errorf("Round(123.456,-5): got %v, want 123", got)
	}
}

func TestToFloat_NonCoercible(t *testing.T) {
	// String that can't parse → 0; bool → 0; nil → 0.
	if toFloat("abc") != 0 {
		t.Errorf("toFloat(abc): got %v", toFloat("abc"))
	}
	if toFloat(true) != 0 {
		t.Errorf("toFloat(true): got %v", toFloat(true))
	}
	if toFloat(nil) != 0 {
		t.Errorf("toFloat(nil): got %v", toFloat(nil))
	}
}
