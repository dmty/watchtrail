package accountctx

import (
	"context"
	"testing"
)

func TestFromDefaultsWhenAbsent(t *testing.T) {
	if got := From(context.Background()); got != Default {
		t.Errorf("From() = %q, want %q", got, Default)
	}
}

func TestRoundTrip(t *testing.T) {
	ctx := With(context.Background(), "acct-42")
	if got := From(ctx); got != "acct-42" {
		t.Errorf("From() = %q, want %q", got, "acct-42")
	}
}

func TestEmptyIDFallsBackToDefault(t *testing.T) {
	ctx := With(context.Background(), "")
	if got := From(ctx); got != Default {
		t.Errorf("From(empty) = %q, want %q", got, Default)
	}
}

func TestDefaultConstant(t *testing.T) {
	if Default != "default" {
		t.Errorf("Default = %q, want %q", Default, "default")
	}
}
