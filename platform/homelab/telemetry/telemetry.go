package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

	"go.opentelemetry.io/contrib/bridges/otelslog"

	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/HeyJIOBUM/homelab-lib/platform/homelab/configloader"
)

type Telemetry struct {
	LogProvider   *sdklog.LoggerProvider
	TraceProvider *sdktrace.TracerProvider
	MeterProvider *sdkmetric.MeterProvider

	config TelemetryConfig
}

type TelemetryConfig struct {
	ServiceName    string
	Environment    string
	LogLevel       string
	TraceEnabled   bool
	MetricsEnabled bool
	SamplingRate   float64
	MetricInterval time.Duration
}

func DefaultConfig() TelemetryConfig {
	return TelemetryConfig{
		LogLevel:       "info",
		TraceEnabled:   false,
		MetricsEnabled: false,
		SamplingRate:   1.0,
		MetricInterval: 10 * time.Second,
	}
}

func LoadConfig() TelemetryConfig {
	return TelemetryConfig{
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		TraceEnabled:   getEnvBool("TRACE_ENABLED", false),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", false),
		SamplingRate:   getEnvFloat("SAMPLING_RATE", 1.0),
		MetricInterval: getEnvDuration("METRIC_INTERVAL", 10*time.Second),
	}
}

func NewTelemetry(appCfg configloader.HomelabAppConfig) (*Telemetry, error) {
	return NewTelemetryWithConfig(appCfg, LoadConfig())
}

func NewTelemetryWithConfig(appCfg configloader.HomelabAppConfig, cfg TelemetryConfig) (*Telemetry, error) {
	cfg.ServiceName = appCfg.AppName
	cfg.Environment = appCfg.AppEnv

	res := sdkresource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.DeploymentEnvironment(cfg.Environment),
	)

	t := &Telemetry{config: cfg}

	logProvider, err := newLoggerProvider(cfg, res)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}
	t.LogProvider = logProvider

	if cfg.TraceEnabled {
		traceProvider, err := newTracerProvider(cfg, res)
		if err != nil {
			_ = t.Shutdown(context.Background())
			return nil, fmt.Errorf("create tracer: %w", err)
		}
		t.TraceProvider = traceProvider
	}

	if cfg.MetricsEnabled {
		meterProvider, err := newMeterProvider(cfg, res)
		if err != nil {
			_ = t.Shutdown(context.Background())
			return nil, fmt.Errorf("create meter: %w", err)
		}
		t.MeterProvider = meterProvider
	}

	return t, nil
}

func (t *Telemetry) NewLogger(scopeName string) *slog.Logger {
	handler := otelslog.NewHandler(
		scopeName,
		otelslog.WithLoggerProvider(t.LogProvider),
	)

	var h slog.Handler = handler
	level := parseLevel(t.config.LogLevel)
	h = &levelHandler{next: h, level: level}

	return slog.New(h)
}

func (t *Telemetry) NewMeter(scopeName string) metric.Meter {
	if t.MeterProvider == nil {
		return metricnoop.NewMeterProvider().Meter(scopeName)
	}
	return t.MeterProvider.Meter(scopeName)
}

func (t *Telemetry) NewTracer(scopeName string) trace.Tracer {
	if t.TraceProvider == nil {
		return tracenoop.NewTracerProvider().Tracer(scopeName)
	}
	return t.TraceProvider.Tracer(scopeName)
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error

	if t.TraceProvider != nil {
		if err := t.TraceProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("trace: %w", err))
		}
		t.TraceProvider = nil
	}

	if t.MeterProvider != nil {
		if err := t.MeterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter: %w", err))
		}
		t.MeterProvider = nil
	}

	if t.LogProvider != nil {
		if err := t.LogProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("log: %w", err))
		}
		t.LogProvider = nil
	}

	return errors.Join(errs...)
}

func newLoggerProvider(_ TelemetryConfig, res *sdkresource.Resource) (*sdklog.LoggerProvider, error) {
	exporter, err := stdoutlog.New(
		stdoutlog.WithPrettyPrint(),
		stdoutlog.WithWriter(os.Stdout),
	)
	if err != nil {
		return nil, fmt.Errorf("create stdout log exporter: %w", err)
	}

	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	return logProvider, nil
}

func newTracerProvider(cfg TelemetryConfig, res *sdkresource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(os.Stdout),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	return provider, nil
}

func newMeterProvider(cfg TelemetryConfig, res *sdkresource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
		stdoutmetric.WithWriter(os.Stdout),
	)
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(cfg.MetricInterval),
		)),
		sdkmetric.WithResource(res),
	)

	return provider, nil
}

type levelHandler struct {
	next  slog.Handler
	level slog.Level
}

func (h *levelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.next.Enabled(ctx, level)
}

func (h *levelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.next.Handle(ctx, r)
}

func (h *levelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelHandler{next: h.next.WithAttrs(attrs), level: h.level}
}

func (h *levelHandler) WithGroup(name string) slog.Handler {
	return &levelHandler{next: h.next.WithGroup(name), level: h.level}
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if val := os.Getenv(key); val != "" {
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			return v
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultValue
}
