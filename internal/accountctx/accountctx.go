// Package accountctx carries the current request's account identifier on
// context.Context. Today the daemon is single-tenant and every request runs
// as Default; the package exists so a future multi-user release can change
// the source of the id (e.g. session lookup) without touching store or
// handler signatures.
package accountctx

import "context"

// Default is the account id used when nothing else is configured. It is the
// only id the single-tenant daemon ever sets.
const Default = "default"

type ctxKey struct{}

// With returns a child context carrying id. An empty id is treated as
// "unset" — From will return Default for it.
func With(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// From extracts the account id from ctx, returning Default if absent.
func From(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return Default
}
