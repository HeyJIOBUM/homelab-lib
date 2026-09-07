package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/multierr"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

	"go.opentelemetry.io/contrib/bridges/otelslog"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/HeyJIOBUM/homelab-lib/platform/homelab/configloader"
)

type Telemetry struct {
	Tracer        trace.Tracer
	Logger        *slog.Logger
	LogProvider   *sdklog.LoggerProvider
	TraceProvider *sdktrace.TracerProvider
	MeterProvider *sdkmetric.MeterProvider
	config        TelemetryConfig
}

type TelemetryConfig struct {
	ServiceName    string
	Environment    string
	LogLevel       string
	LogFormat      string
	TraceEnabled   bool
	MetricsEnabled bool
	SamplingRate   float64
	MetricInterval time.Duration
}

func DefaultConfig() TelemetryConfig {
	return TelemetryConfig{
		LogLevel:       "info",
		LogFormat:      "json",
		TraceEnabled:   false,
		MetricsEnabled: false,
		SamplingRate:   1.0,
		MetricInterval: 10 * time.Second,
	}
}

func LoadConfig() TelemetryConfig {
	return TelemetryConfig{
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		LogFormat:      getEnv("LOG_FORMAT", "json"),
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

	logger, logProvider, err := newLogger(cfg, res)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	var tracer trace.Tracer
	var traceProvider *sdktrace.TracerProvider
	var meterProvider *sdkmetric.MeterProvider

	if cfg.TraceEnabled {
		tracer, traceProvider, err = newTracer(res, cfg)
		if err != nil {
			return nil, fmt.Errorf("create tracer: %w", err)
		}
	} else {
		tracer = tracenoop.NewTracerProvider().Tracer(cfg.ServiceName)
	}

	if cfg.MetricsEnabled {
		meterProvider, err = newMeter(res, cfg)
		if err != nil {
			return nil, fmt.Errorf("create meter: %w", err)
		}
	}

	return &Telemetry{
		Logger:        logger,
		LogProvider:   logProvider,
		Tracer:        tracer,
		TraceProvider: traceProvider,
		MeterProvider: meterProvider,
		config:        cfg,
	}, nil
}

func newLogger(cfg TelemetryConfig, res *sdkresource.Resource) (*slog.Logger, *sdklog.LoggerProvider, error) {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetLogLoggerLevel(level)

	exporter, err := stdoutlog.New(
		stdoutlog.WithPrettyPrint(),
		stdoutlog.WithWriter(os.Stdout),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create stdout log exporter: %w", err)
	}

	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	logger := slog.New(otelslog.NewHandler(
		cfg.ServiceName,
		otelslog.WithLoggerProvider(logProvider),
	))

	return logger, logProvider, nil
}

func newTracer(res *sdkresource.Resource, cfg TelemetryConfig) (trace.Tracer, *sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(os.Stdout),
	)
	if err != nil {
		return nil, nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return otel.Tracer(cfg.ServiceName), provider, nil
}

func newMeter(res *sdkresource.Resource, cfg TelemetryConfig) (*sdkmetric.MeterProvider, error) {
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

	otel.SetMeterProvider(provider)
	return provider, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs error

	if t.TraceProvider != nil {
		if err := t.TraceProvider.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("trace: %w", err))
		}
	}

	if t.MeterProvider != nil {
		if err := t.MeterProvider.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("meter: %w", err))
		}
	}

	if t.LogProvider != nil {
		if err := t.LogProvider.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("log: %w", err))
		}
	}

	return errs
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
		var v float64
		fmt.Sscanf(val, "%f", &v)
		return v
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
