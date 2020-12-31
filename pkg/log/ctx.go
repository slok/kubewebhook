package log

import (
	"context"
)

type contextKey string

// contextLogValuesKey used as unique key to store log values in the context.
const contextLogValuesKey = contextKey("kubewebhook-log-values")

// CtxWithValues returns a copy of parent in which the key values passed have been
// stored ready to be used using log.Logger.
func CtxWithValues(parent context.Context, kv Kv) context.Context {
	// Maybe we have values already set.
	oldValues, ok := parent.Value(contextLogValuesKey).(Kv)
	if !ok {
		oldValues = Kv{}
	}

	// Copy old and received values into the new kv.
	newValues := Kv{}
	for k, v := range oldValues {
		newValues[k] = v
	}
	for k, v := range kv {
		newValues[k] = v
	}

	return context.WithValue(parent, contextLogValuesKey, newValues)
}

// ValuesFromCtx gets the log Key values from a context.
func ValuesFromCtx(ctx context.Context) Kv {
	values, ok := ctx.Value(contextLogValuesKey).(Kv)
	if !ok {
		return Kv{}
	}

	return values
}
