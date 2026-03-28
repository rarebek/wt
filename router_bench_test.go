package wt

import "testing"

func BenchmarkRouterMatch1Route(b *testing.B) {
	r := NewRouter()
	r.Add("/echo", func(c *Context) {})

	b.ResetTimer()
	for b.Loop() {
		r.Match("/echo")
	}
}

func BenchmarkRouterMatch10Routes(b *testing.B) {
	r := NewRouter()
	for i := range 10 {
		path := "/" + string(rune('a'+i))
		r.Add(path+"/{id}", func(c *Context) {})
	}

	b.ResetTimer()
	for b.Loop() {
		r.Match("/e/42") // middle of the list
	}
}

func BenchmarkRouterMatch50Routes(b *testing.B) {
	r := NewRouter()
	for i := range 50 {
		path := "/" + string(rune('a'+(i%26))) + string(rune('a'+(i/26)))
		r.Add(path+"/{id}", func(c *Context) {})
	}

	b.ResetTimer()
	for b.Loop() {
		r.Match("/y/42") // near end
	}
}

func BenchmarkRouterMatchCatchAll(b *testing.B) {
	r := NewRouter()
	r.Add("/static/{path...}", func(c *Context) {})

	b.ResetTimer()
	for b.Loop() {
		r.Match("/static/css/main.css")
	}
}
