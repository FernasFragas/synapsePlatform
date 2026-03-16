package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"synapsePlatform/internal/auth"
)

type AuthValidator struct {
	validator auth.TokenValidator
	tracer    trace.Tracer
	duration  metric.Float64Histogram
	total     metric.Int64Counter
	errors    metric.Int64Counter
}

func NewAuthValidator(meter metric.Meter, tracer trace.Tracer, validator auth.TokenValidator) (*AuthValidator, error) {
	duration, err := meter.Float64Histogram("auth.validate.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time to validate a JWT token"),
	)
	if err != nil {
		return nil, err
	}

	total, err := meter.Int64Counter("auth.validate.total",
		metric.WithDescription("Total token validations by status"),
	)
	if err != nil {
		return nil, err
	}

	errors, err := meter.Int64Counter("auth.validate.errors",
		metric.WithDescription("Token validation failures"),
	)
	if err != nil {
		return nil, err
	}

	return &AuthValidator{
		validator: validator,
		tracer:    tracer,
		duration:  duration,
		total:     total,
		errors:    errors,
	}, nil
}

func (m *AuthValidator) Validate(tokenString string) (auth.Identity, error) {
	ctx, span := m.tracer.Start(context.Background(), "auth.validate")
	defer span.End()

	start := time.Now()

	identity, err := m.validator.Validate(tokenString)

	elapsed := time.Since(start).Seconds()

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		m.errors.Add(ctx, 1, metric.WithAttributes(
			attribute.String(AttrOperation, "validate"),
		))
		m.total.Add(ctx, 1, metric.WithAttributes(
			attribute.String(AttrOperation, "validate"),
			attribute.String(AttrStatus, StatusError),
		))
		m.duration.Record(ctx, elapsed, metric.WithAttributes(
			attribute.String(AttrStatus, StatusError),
		))

		return auth.Identity{}, err
	}

	span.SetAttributes(
		attribute.String("subject", identity.Subject),
		attribute.String("client_id", identity.ClientID),
	)

	m.total.Add(ctx, 1, metric.WithAttributes(
		attribute.String(AttrOperation, "validate"),
		attribute.String(AttrStatus, StatusSuccess),
	))
	m.duration.Record(ctx, elapsed, metric.WithAttributes(
		attribute.String(AttrStatus, StatusSuccess),
	))

	return identity, nil
}
