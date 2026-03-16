package metrics

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Providers struct {
	Meter   *sdkmetric.MeterProvider
	Tracer  *sdktrace.TracerProvider
	Metrics http.Handler
}

func NewProviders(ctx context.Context, serviceName, traceEndpoint string) (*Providers, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, err
	}

	registry := promclient.NewRegistry()

	promExporter, err := prometheus.New(prometheus.WithRegisterer(registry))
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	)

	tracerProvider, err := buildTracerProvider(ctx, traceEndpoint, res)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Providers{
		Meter:   meterProvider,
		Tracer:  tracerProvider,
		Metrics: promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}, nil
}

func (p *Providers) Shutdown(ctx context.Context) error {
	if err := p.Tracer.Shutdown(ctx); err != nil {
		return err
	}
	return p.Meter.Shutdown(ctx)
}

func buildTracerProvider(ctx context.Context, endpoint string, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	switch endpoint {
	case "stdout":
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		return sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(res),
		), nil
	case "":
		return sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		), nil
	default:
		exporter, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(endpoint),
		)
		if err != nil {
			return nil, err
		}
		return sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		), nil
	}
}
