package request

import "testing"

func TestResolveAuthChain(t *testing.T) {
	bearer := AuthSpec{Type: "bearer", Config: map[string]any{"token": "root"}}
	basic := AuthSpec{Type: "basic", Config: map[string]any{"username": "child"}}

	t.Run("request explicit auth wins", func(t *testing.T) {
		got := ResolveAuthChain(AuthSpec{Type: "apikey"}, []AuthSpec{bearer})
		if got.Type != "apikey" {
			t.Fatalf("got %q want apikey", got.Type)
		}
	})

	t.Run("request inherit uses nearest folder", func(t *testing.T) {
		got := ResolveAuthChain(AuthSpec{Type: "inherit"}, []AuthSpec{basic, bearer})
		if got.Type != "basic" {
			t.Fatalf("got %q want basic", got.Type)
		}
	})

	t.Run("request inherit skips inherit folders", func(t *testing.T) {
		got := ResolveAuthChain(AuthSpec{Type: "inherit"}, []AuthSpec{{Type: "inherit"}, bearer})
		if got.Type != "bearer" {
			t.Fatalf("got %q want bearer", got.Type)
		}
	})

	t.Run("request inherit with no explicit auth returns none", func(t *testing.T) {
		got := ResolveAuthChain(AuthSpec{Type: "inherit"}, []AuthSpec{{Type: "inherit"}})
		if got.Type != "none" {
			t.Fatalf("got %q want none", got.Type)
		}
	})

	t.Run("request none does not inherit", func(t *testing.T) {
		got := ResolveAuthChain(AuthSpec{Type: "none"}, []AuthSpec{bearer})
		if got.Type != "none" {
			t.Fatalf("got %q want none", got.Type)
		}
	})
}
