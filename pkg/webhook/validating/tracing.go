package validating

import (
	"context"

	opentracing "github.com/opentracing/opentracing-go"
	opentracingext "github.com/opentracing/opentracing-go/ext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TraceValidator will wrap the validator and trace the received validator. for example this helper
// can be used to trace each of the validators and get what parts of the validating chain is the
// bottleneck.
func TraceValidator(tracer opentracing.Tracer, validatorName string, m Validator) Validator {
	if tracer == nil {
		tracer = &opentracing.NoopTracer{}
	}
	return &tracedValidator{
		validator:     m,
		tracer:        tracer,
		validatorName: validatorName,
	}
}

type tracedValidator struct {
	validator     Validator
	validatorName string
	tracer        opentracing.Tracer
}

func (m *tracedValidator) Validate(ctx context.Context, obj metav1.Object) (stop bool, valid ValidatorResult, err error) {
	span, ctx := m.createValidatorSpan(ctx)
	defer span.Finish()

	span.LogKV("event", "start_validate")

	// Validate.
	stop, res, err := m.validator.Validate(ctx, obj)

	if err != nil {
		opentracingext.Error.Set(span, true)
		span.LogKV(
			"event", "error",
			"message", err,
		)
		return stop, res, err
	}

	span.LogKV(
		"event", "end_validate",
		"stopChain", stop,
		"valid", res.Valid,
		"message", res.Message,
	)

	return stop, res, nil
}

func (m *tracedValidator) createValidatorSpan(ctx context.Context) (opentracing.Span, context.Context) {
	var spanOpts []opentracing.StartSpanOption

	// Check if we receive a previous span or we are the root span.
	if pSpan := opentracing.SpanFromContext(ctx); pSpan != nil {
		spanOpts = append(spanOpts, opentracing.ChildOf(pSpan.Context()))
	}

	// Create a new span.
	span := m.tracer.StartSpan("validate", spanOpts...)

	// Set span data.
	span.SetTag("kubewebhook.validator.name", m.validatorName)

	ctx = opentracing.ContextWithSpan(ctx, span)
	return span, ctx
}
