package test

import (
	"context"
	"time"
)

// Context is a context that is also a [TestingT].
type Context interface {
	context.Context
	TestingT
}

// WithContext returns a context that is bound to the lifetime of the test.
func WithContext(t TestingT) Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	return &testContext{
		Context:  ctx,
		TestingT: t,
	}
}

type testContext struct {
	context.Context
	TestingT
}

func contextOf(t TestingT) context.Context {
	if t, ok := t.(Context); ok {
		return t
	}

	return context.Background()
}
