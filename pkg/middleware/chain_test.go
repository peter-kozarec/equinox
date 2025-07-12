package middleware

import (
	"testing"
)

func TestMiddleware_Chain(t *testing.T) {
	type handler func(int) int

	add10 := func(h handler) handler {
		return func(n int) int {
			return h(n) + 10
		}
	}

	multiply2 := func(h handler) handler {
		return func(n int) int {
			return h(n) * 2
		}
	}

	base := func(n int) int {
		return n
	}

	chained := Chain(add10, multiply2)(base)
	result := chained(5)

	if result != 20 {
		t.Errorf("Expected 20, got %d", result)
	}
}

func TestMiddleware_ChainEmpty(t *testing.T) {
	type handler func(string) string

	base := func(s string) string {
		return s
	}

	chained := Chain[handler]()(base)
	result := chained("test")

	if result != "test" {
		t.Errorf("Expected 'test', got %s", result)
	}
}

func TestMiddleware_ChainSingle(t *testing.T) {
	type handler func(string) string

	uppercase := func(h handler) handler {
		return func(s string) string {
			return h(s) + "!"
		}
	}

	base := func(s string) string {
		return s
	}

	chained := Chain(uppercase)(base)
	result := chained("hello")

	if result != "hello!" {
		t.Errorf("Expected 'hello!', got %s", result)
	}
}

func TestMiddleware_ChainOrder(t *testing.T) {
	type handler func([]string) []string

	appendA := func(h handler) handler {
		return func(s []string) []string {
			result := h(s)
			return append(result, "A")
		}
	}

	appendB := func(h handler) handler {
		return func(s []string) []string {
			result := h(s)
			return append(result, "B")
		}
	}

	appendC := func(h handler) handler {
		return func(s []string) []string {
			result := h(s)
			return append(result, "C")
		}
	}

	base := func(s []string) []string {
		return append(s, "base")
	}

	chained := Chain(appendA, appendB, appendC)(base)
	result := chained([]string{})

	expected := []string{"base", "C", "B", "A"}
	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("At index %d: expected %s, got %s", i, expected[i], v)
		}
	}
}

func TestMiddleware_ChainStruct(t *testing.T) {
	type Request struct {
		Value  int
		Method string
	}

	type handler func(Request) Request

	addValue := func(h handler) handler {
		return func(r Request) Request {
			result := h(r)
			result.Value += 5
			return result
		}
	}

	setMethod := func(h handler) handler {
		return func(r Request) Request {
			result := h(r)
			result.Method = "POST"
			return result
		}
	}

	base := func(r Request) Request {
		r.Value *= 2
		return r
	}

	chained := Chain(addValue, setMethod)(base)
	result := chained(Request{Value: 10, Method: "GET"})

	if result.Value != 25 {
		t.Errorf("Expected Value=25, got %d", result.Value)
	}

	if result.Method != "POST" {
		t.Errorf("Expected Method='POST', got %s", result.Method)
	}
}

func TestMiddleware_ChainMultipleParameters(t *testing.T) {
	type handler func(int, int) (int, int)

	swap := func(h handler) handler {
		return func(a, b int) (int, int) {
			return h(b, a)
		}
	}

	add1ToBoth := func(h handler) handler {
		return func(a, b int) (int, int) {
			x, y := h(a, b)
			return x + 1, y + 1
		}
	}

	base := func(a, b int) (int, int) {
		return a * 2, b * 3
	}

	chained := Chain(swap, add1ToBoth)(base)
	x, y := chained(5, 10)

	if x != 21 {
		t.Errorf("Expected x=21, got %d", x)
	}

	if y != 16 {
		t.Errorf("Expected y=16, got %d", y)
	}
}

func TestMiddleware_ChainError(t *testing.T) {
	type handler func(int) (int, error)

	checkPositive := func(h handler) handler {
		return func(n int) (int, error) {
			if n < 0 {
				return 0, &testError{"negative value"}
			}
			return h(n)
		}
	}

	doubleValue := func(h handler) handler {
		return func(n int) (int, error) {
			val, err := h(n)
			if err != nil {
				return 0, err
			}
			return val * 2, nil
		}
	}

	base := func(n int) (int, error) {
		return n + 10, nil
	}

	chained := Chain(checkPositive, doubleValue)(base)

	result, err := chained(5)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 30 {
		t.Errorf("Expected 30, got %d", result)
	}

	_, err = chained(-5)
	if err == nil {
		t.Error("Expected error for negative value")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestMiddleware_ChainState(t *testing.T) {
	type handler func() int

	counter := 0

	increment := func(h handler) handler {
		return func() int {
			counter++
			return h() + counter
		}
	}

	base := func() int {
		return 0
	}

	chained := Chain(increment, increment, increment)(base)

	result1 := chained()
	if result1 != 9 {
		t.Errorf("First call: expected 9, got %d", result1)
	}

	result2 := chained()
	if result2 != 18 {
		t.Errorf("Second call: expected 18, got %d", result2)
	}
}

func TestMiddleware_ChainLarge(t *testing.T) {
	type handler func(int) int

	increment := func(h handler) handler {
		return func(n int) int {
			return h(n) + 1
		}
	}

	base := func(n int) int {
		return n
	}

	var middlewares []func(handler) handler
	for i := 0; i < 100; i++ {
		middlewares = append(middlewares, increment)
	}

	chained := Chain(middlewares...)(base)
	result := chained(0)

	if result != 100 {
		t.Errorf("Expected 100, got %d", result)
	}
}

func BenchmarkMiddleware_Chain(b *testing.B) {
	type handler func(int) int

	add := func(n int) func(handler) handler {
		return func(h handler) handler {
			return func(x int) int {
				return h(x) + n
			}
		}
	}

	base := func(n int) int {
		return n
	}

	chained := Chain(add(1), add(2), add(3))(base)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chained(0)
	}
}

func BenchmarkMiddleware_ChainDeep(b *testing.B) {
	type handler func(int) int

	passthrough := func(h handler) handler {
		return func(n int) int {
			return h(n)
		}
	}

	base := func(n int) int {
		return n
	}

	var middlewares []func(handler) handler
	for i := 0; i < 50; i++ {
		middlewares = append(middlewares, passthrough)
	}

	chained := Chain(middlewares...)(base)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chained(42)
	}
}

func BenchmarkMiddleware_ChainEmpty(b *testing.B) {
	type handler func(int) int

	base := func(n int) int {
		return n * 2
	}

	chained := Chain[handler]()(base)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chained(42)
	}
}

func BenchmarkMiddleware_ChainComplex(b *testing.B) {
	type Request struct {
		ID     int
		Data   string
		Values []int
	}

	type handler func(Request) Request

	transform := func(h handler) handler {
		return func(r Request) Request {
			result := h(r)
			result.ID++
			result.Data += "x"
			result.Values = append(result.Values, len(result.Values))
			return result
		}
	}

	base := func(r Request) Request {
		return r
	}

	chained := Chain(transform, transform, transform)(base)

	req := Request{
		ID:     1,
		Data:   "test",
		Values: []int{1, 2, 3},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chained(req)
	}
}
