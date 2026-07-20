# acme-std-math

Math primitives for WSL workflows. Provides the basic arithmetic and rounding operations that business logic needs (invoice line computation, tax calc, quantity math) without embedding Go in transition methods.

All inputs are `interface{}` to avoid the WSL int/float64 JSON-decoding panic (a Go method with `int` or `float64` params panics when called from WSL with a JSON number). Outputs are `float64` — if the caller needs integer or string formatting, use WSL `|int` / `|string` filters.

## WSL usage

```wsl
import math/ops

state Compute {
  action math/ops.Mul(a: $qty, b: $price) as line
  action math/ops.Round(value: $line.result, decimals: 2) as rounded
}
```

## Actions

| Action | Params | Returns |
|---|---|---|
| `Add` | `a`, `b` | `{"result": float64}` |
| `Sub` | `a`, `b` | `{"result": float64}` |
| `Mul` | `a`, `b` | `{"result": float64}` |
| `Div` | `a`, `b` | `{"result": float64}` — `400 division_by_zero` if b == 0 |
| `Round` | `value`, `decimals int` | `{"result": float64}` |
| `SumArray` | `values []interface{}` | `{"result": float64}` |
| `Abs` | `value` | `{"result": float64}` |
| `Min` | `a`, `b` | `{"result": float64}` |
| `Max` | `a`, `b` | `{"result": float64}` |

## Why interface{}

WSL passes JSON numbers to Go methods. The engine's reflection-based dispatch decodes these as `float64` (or `int` for whole numbers). A Go method declaring `int` panics if WSL passes `3.14`; declaring `float64` panics if WSL passes `3` (integer-literal). `interface{}` + runtime coercion is the only safe contract. See `wsl_int_numeric_args.md` memory.
