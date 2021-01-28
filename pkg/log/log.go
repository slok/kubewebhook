package log

import "context"

// Kv is a helper type for structured logging fields usage.
type Kv = map[string]interface{}

// Logger is the interface that the loggers used by the library will use.
type Logger interface {
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	WithValues(values map[string]interface{}) Logger
	WithCtxValues(ctx context.Context) Logger
	SetValuesOnCtx(parent context.Context, values map[string]interface{}) context.Context
}

// Noop logger doesn't log anything.
const Noop = noop(0)

type noop int

func (n noop) Infof(format string, args ...interface{})                         {}
func (n noop) Warningf(format string, args ...interface{})                      {}
func (n noop) Errorf(format string, args ...interface{})                        {}
func (n noop) Debugf(format string, args ...interface{})                        {}
func (n noop) WithValues(map[string]interface{}) Logger                         { return n }
func (n noop) WithCtxValues(context.Context) Logger                             { return n }
func (n noop) SetValuesOnCtx(parent context.Context, values Kv) context.Context { return parent }
