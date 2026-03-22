package observability

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

const (
	ErrorTypeValidation     = "validation_error"
	ErrorTypeDomain         = "domain_error"
	ErrorTypeIntegration    = "integration_error"
	ErrorTypeInfrastructure = "infrastructure_error"
)

type Config struct {
	ServiceName   string
	Environment   string
	TraceEndpoint string
	TraceExporter sdktrace.SpanExporter
}

type Runtime struct {
	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
	promHandler    http.Handler
	propagator     propagation.TextMapPropagator
	queryTracer    pgx.QueryTracer

	httpRequestsTotal      metric.Int64Counter
	httpRequestErrorsTotal metric.Int64Counter
	httpRequestDuration    metric.Float64Histogram
	eventsPublishedTotal   metric.Int64Counter
	eventsConsumedTotal    metric.Int64Counter
	eventHandlerFailures   metric.Int64Counter
	dbQueryDuration        metric.Float64Histogram
	gatewayRequestDuration metric.Float64Histogram
	gatewayFailuresTotal   metric.Int64Counter
	paymentRetriesTotal    metric.Int64Counter
	shutdownTraceProvider  func(context.Context) error
	shutdownMeterProvider  func(context.Context) error
}

func New(ctx context.Context, cfg Config) (*Runtime, error) {
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "atlas-erp-core"
	}

	environment := strings.TrimSpace(cfg.Environment)
	if environment == "" {
		environment = "local"
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("deployment.environment", environment),
	))
	if err != nil {
		return nil, fmt.Errorf("build telemetry resource: %w", err)
	}

	registry := prometheus.NewRegistry()
	promExporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
		otelprom.WithoutCounterSuffixes(),
		otelprom.WithoutScopeInfo(),
		otelprom.WithoutTargetInfo(),
		otelprom.WithoutUnits(),
	)
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
	)

	traceOptions := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}

	if cfg.TraceExporter != nil {
		traceOptions = append(traceOptions, sdktrace.WithSyncer(cfg.TraceExporter))
	}

	if endpoint := strings.TrimSpace(cfg.TraceEndpoint); endpoint != "" {
		exporter, exporterErr := newTraceExporter(ctx, endpoint)
		if exporterErr != nil {
			return nil, exporterErr
		}

		traceOptions = append(traceOptions, sdktrace.WithBatcher(exporter))
	}

	traceProvider := sdktrace.NewTracerProvider(traceOptions...)

	runtime := &Runtime{
		tracerProvider: traceProvider,
		meterProvider:  meterProvider,
		promHandler: promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}),
		propagator: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
		shutdownTraceProvider: traceProvider.Shutdown,
		shutdownMeterProvider: meterProvider.Shutdown,
	}

	if err := runtime.initMetrics(); err != nil {
		return nil, err
	}

	runtime.queryTracer = newQueryTracer(runtime)

	return runtime, nil
}

func NewNoop() *Runtime {
	runtime, err := New(context.Background(), Config{
		ServiceName: "atlas-erp-core",
		Environment: "test",
	})
	if err != nil {
		panic(err)
	}

	return runtime
}

func FromOptional(runtimes ...*Runtime) *Runtime {
	for _, runtime := range runtimes {
		if runtime != nil {
			return runtime
		}
	}

	return NewNoop()
}

func (runtime *Runtime) Shutdown(ctx context.Context) error {
	if runtime == nil {
		return nil
	}

	var shutdownErr error
	if runtime.shutdownMeterProvider != nil {
		if err := runtime.shutdownMeterProvider(ctx); err != nil {
			shutdownErr = err
		}
	}

	if runtime.shutdownTraceProvider != nil {
		if err := runtime.shutdownTraceProvider(ctx); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
	}

	return shutdownErr
}

func (runtime *Runtime) MetricsHandler() http.Handler {
	if runtime == nil || runtime.promHandler == nil {
		return promhttp.Handler()
	}

	return runtime.promHandler
}

func (runtime *Runtime) TracerProvider() trace.TracerProvider {
	if runtime == nil || runtime.tracerProvider == nil {
		return trace.NewNoopTracerProvider()
	}

	return runtime.tracerProvider
}

func (runtime *Runtime) MeterProvider() metric.MeterProvider {
	if runtime == nil || runtime.meterProvider == nil {
		return noopmetric.NewMeterProvider()
	}

	return runtime.meterProvider
}

func (runtime *Runtime) Propagator() propagation.TextMapPropagator {
	if runtime == nil || runtime.propagator == nil {
		return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}

	return runtime.propagator
}

func (runtime *Runtime) Tracer(name string) trace.Tracer {
	return runtime.TracerProvider().Tracer(name)
}

func (runtime *Runtime) QueryTracer() pgx.QueryTracer {
	if runtime == nil {
		return nil
	}

	return runtime.queryTracer
}

func (runtime *Runtime) StartUseCase(ctx context.Context, module string, useCase string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	baseAttributes := []attribute.KeyValue{
		attribute.String("atlas.module", module),
		attribute.String("atlas.use_case", useCase),
	}

	return runtime.Tracer("atlas-erp-core/application").Start(
		ctx,
		fmt.Sprintf("application.usecase %s.%s", module, useCase),
		trace.WithAttributes(append(baseAttributes, attrs...)...),
	)
}

func (runtime *Runtime) StartIntegration(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return runtime.Tracer("atlas-erp-core/integration").Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
}

func (runtime *Runtime) StartEventPublish(ctx context.Context, eventName string, emitterModule string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	baseAttributes := []attribute.KeyValue{
		attribute.String("atlas.event_name", eventName),
		attribute.String("atlas.emitter_module", emitterModule),
	}

	return runtime.Tracer("atlas-erp-core/event").Start(
		ctx,
		fmt.Sprintf("event.publish %s", eventName),
		trace.WithAttributes(append(baseAttributes, attrs...)...),
	)
}

func (runtime *Runtime) StartEventConsume(ctx context.Context, eventName string, consumerModule string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	baseAttributes := []attribute.KeyValue{
		attribute.String("atlas.event_name", eventName),
		attribute.String("atlas.consumer_module", consumerModule),
	}

	return runtime.Tracer("atlas-erp-core/event").Start(
		ctx,
		fmt.Sprintf("event.consume %s.%s", consumerModule, eventName),
		trace.WithAttributes(append(baseAttributes, attrs...)...),
	)
}

func (runtime *Runtime) CompleteSpan(span trace.Span, err error, errorType string) {
	if span == nil {
		return
	}
	defer span.End()

	if err == nil {
		span.SetStatus(codes.Ok, "")
		return
	}

	runtime.RecordSpanError(span, err, errorType)
}

func (runtime *Runtime) RecordSpanError(span trace.Span, err error, errorType string) {
	if span == nil || err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	if strings.TrimSpace(errorType) != "" {
		span.SetAttributes(attribute.String("error.type", errorType))
	}
}

func (runtime *Runtime) RecordHTTPRequest(ctx context.Context, method string, route string, statusCode int, module string, errorType string, duration time.Duration) {
	if runtime == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("http.method", normalizeMetricValue(method, "UNKNOWN")),
		attribute.String("http.route", normalizeMetricValue(route, "unknown")),
		attribute.String("http.status_code", fmt.Sprintf("%d", statusCode)),
		attribute.String("atlas.module", normalizeMetricValue(module, "shared")),
	}

	runtime.httpRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	runtime.httpRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if statusCode >= http.StatusBadRequest {
		errorAttrs := append([]attribute.KeyValue{}, attrs...)
		errorAttrs = append(errorAttrs, attribute.String("error.type", normalizeMetricValue(errorType, ErrorTypeInfrastructure)))
		runtime.httpRequestErrorsTotal.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
	}
}

func (runtime *Runtime) RecordEventPublished(ctx context.Context, eventName string, emitterModule string) {
	if runtime == nil {
		return
	}

	runtime.eventsPublishedTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("atlas.event_name", normalizeMetricValue(eventName, "unknown")),
		attribute.String("atlas.emitter_module", normalizeMetricValue(emitterModule, "shared")),
	))
}

func (runtime *Runtime) RecordEventConsumed(ctx context.Context, eventName string, consumerModule string) {
	if runtime == nil {
		return
	}

	runtime.eventsConsumedTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("atlas.event_name", normalizeMetricValue(eventName, "unknown")),
		attribute.String("atlas.consumer_module", normalizeMetricValue(consumerModule, "shared")),
	))
}

func (runtime *Runtime) RecordEventHandlerFailure(ctx context.Context, eventName string, consumerModule string, errorType string) {
	if runtime == nil {
		return
	}

	runtime.eventHandlerFailures.Add(ctx, 1, metric.WithAttributes(
		attribute.String("atlas.event_name", normalizeMetricValue(eventName, "unknown")),
		attribute.String("atlas.consumer_module", normalizeMetricValue(consumerModule, "shared")),
		attribute.String("error.type", normalizeMetricValue(errorType, ErrorTypeInfrastructure)),
	))
}

func (runtime *Runtime) RecordDBQuery(ctx context.Context, operation string, table string, duration time.Duration) {
	if runtime == nil {
		return
	}

	runtime.dbQueryDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("db.operation", normalizeMetricValue(operation, "query")),
		attribute.String("db.sql.table", normalizeMetricValue(table, "unknown")),
	))
}

func (runtime *Runtime) RecordGatewayRequest(ctx context.Context, duration time.Duration) {
	if runtime == nil {
		return
	}

	runtime.gatewayRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("atlas.module", "payments"),
	))
}

func (runtime *Runtime) RecordGatewayFailure(ctx context.Context, errorType string) {
	if runtime == nil {
		return
	}

	runtime.gatewayFailuresTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("atlas.module", "payments"),
		attribute.String("error.type", normalizeMetricValue(errorType, ErrorTypeIntegration)),
	))
}

func (runtime *Runtime) RecordPaymentRetry(ctx context.Context) {
	if runtime == nil {
		return
	}

	runtime.paymentRetriesTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("atlas.module", "payments"),
	))
}

func (runtime *Runtime) initMetrics() error {
	meter := runtime.MeterProvider().Meter("atlas-erp-core/observability")

	var err error
	if runtime.httpRequestsTotal, err = meter.Int64Counter("atlas_erp_http_requests_total"); err != nil {
		return fmt.Errorf("create http request counter: %w", err)
	}

	if runtime.httpRequestErrorsTotal, err = meter.Int64Counter("atlas_erp_http_request_errors_total"); err != nil {
		return fmt.Errorf("create http request error counter: %w", err)
	}

	if runtime.httpRequestDuration, err = meter.Float64Histogram("atlas_erp_http_request_duration_seconds"); err != nil {
		return fmt.Errorf("create http request duration histogram: %w", err)
	}

	if runtime.eventsPublishedTotal, err = meter.Int64Counter("atlas_erp_events_published_total"); err != nil {
		return fmt.Errorf("create events published counter: %w", err)
	}

	if runtime.eventsConsumedTotal, err = meter.Int64Counter("atlas_erp_events_consumed_total"); err != nil {
		return fmt.Errorf("create events consumed counter: %w", err)
	}

	if runtime.eventHandlerFailures, err = meter.Int64Counter("atlas_erp_event_handler_failures_total"); err != nil {
		return fmt.Errorf("create event handler failures counter: %w", err)
	}

	if runtime.dbQueryDuration, err = meter.Float64Histogram("atlas_erp_db_query_duration_seconds"); err != nil {
		return fmt.Errorf("create db query duration histogram: %w", err)
	}

	if runtime.gatewayRequestDuration, err = meter.Float64Histogram("atlas_erp_gateway_request_duration_seconds"); err != nil {
		return fmt.Errorf("create gateway duration histogram: %w", err)
	}

	if runtime.gatewayFailuresTotal, err = meter.Int64Counter("atlas_erp_gateway_failures_total"); err != nil {
		return fmt.Errorf("create gateway failures counter: %w", err)
	}

	if runtime.paymentRetriesTotal, err = meter.Int64Counter("atlas_erp_payment_retries_total"); err != nil {
		return fmt.Errorf("create payment retries counter: %w", err)
	}

	return nil
}

func newTraceExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return nil, fmt.Errorf("parse OTEL_EXPORTER_OTLP_ENDPOINT: %w", err)
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(parsedURL.String()),
	}

	if parsedURL.Scheme == "http" {
		options = append(options, otlptracehttp.WithInsecure())
	}

	if parsedURL.Path == "" || parsedURL.Path == "/" {
		options = append(options, otlptracehttp.WithURLPath("/v1/traces"))
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	return exporter, nil
}

func normalizeMetricValue(value string, fallback string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return fallback
	}

	return normalized
}
