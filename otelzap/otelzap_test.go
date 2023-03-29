package otelzap

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap/zaptest/observer"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type Test struct {
	log     func(ctx context.Context, log *Logger)
	require func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs)
}

func TestOtelZap(t *testing.T) {
	tests := []Test{
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Info("hello")
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "INFO", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.InfoContext(ctx, "hello")
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "INFO", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Warn("hello", zap.String("foo", "bar"))
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, "bar", foo.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Warn("hello", zap.Strings("foo", []string{"bar1", "bar2", "bar3"}))
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, []string{"bar1", "bar2", "bar3"}, foo.AsStringSlice())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).
					WithOptions(zap.Fields(zap.String("baz", "baz1"))).
					WithOptions(zap.Fields(zap.String("faz", "faz1"))).
					Warn("hello", zap.Strings("foo", []string{"bar1", "bar2", "bar3"}))
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, []string{"bar1", "bar2", "bar3"}, foo.AsStringSlice())

				baz, ok := m["baz"]
				require.True(t, ok)
				require.Equal(t, "baz1", baz.AsString())

				faz, ok := m["faz"]
				require.True(t, ok)
				require.Equal(t, "faz1", faz.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Ctx(ctx).Warn("hello", zap.Durations("foo", []time.Duration{time.Millisecond, time.Second, time.Hour}))
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "WARN", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.Equal(t, []string{"1ms", "1s", "1h0m0s"}, foo.AsStringSlice())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				err := errors.New("some error")
				log.Ctx(ctx).Error("hello", zap.Error(err))
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				excTyp, ok := m[semconv.ExceptionTypeKey]
				require.True(t, ok)
				require.Equal(t, "*errors.errorString", excTyp.AsString())

				excMsg, ok := m[semconv.ExceptionMessageKey]
				require.True(t, ok)
				require.Equal(t, "some error", excMsg.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log = log.Clone(WithStackTrace(true))
				log.Ctx(ctx).Info("hello")
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				stack, ok := m[semconv.ExceptionStacktraceKey]
				require.True(t, ok)
				require.NotZero(t, stack.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().ErrorwContext(ctx, "hello", "foo", "bar")
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.NotZero(t, foo.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().ErrorfContext(ctx, "hello %s", "world")
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello world", msg.AsString())

				tpl, ok := m[logTemplateKey]
				require.True(t, ok)
				require.Equal(t, "hello %s", tpl.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().Ctx(ctx).Errorw("hello", "foo", "bar")
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.NotZero(t, foo.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().InfowContext(ctx, "hello", "foo", "bar")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "INFO", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				foo, ok := m["foo"]
				require.True(t, ok)
				require.NotZero(t, foo.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().InfowContext(ctx, "sugary logs require keyAndValues to come in pairs", "so this is invalid, but it shouldn't panic")
			},
			require: func(t *testing.T, event sdktrace.Event) {
				// no panic? success!
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log.Sugar().Ctx(ctx).Errorf("hello %s", "world")
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "ERROR", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello world", msg.AsString())

				tpl, ok := m[logTemplateKey]
				require.True(t, ok)
				require.Equal(t, "hello %s", tpl.AsString())

				requireCodeAttrs(t, m)
			},
		},
		{
			log: func(ctx context.Context, log *Logger) {
				log = log.Clone(WithSetTraceFieldsFunc(func(spanCtx trace.SpanContext) []zapcore.Field {
					return []zapcore.Field{zap.String("MyTraceId", "123"), zap.String("MySpanID", "456")}
				}))
				log.Ctx(ctx).Info("hello")
			},
			require: func(t *testing.T, event sdktrace.Event, logs *observer.ObservedLogs) {
				m := attrMap(event.Attributes)

				sev, ok := m[logSeverityKey]
				require.True(t, ok)
				require.Equal(t, "INFO", sev.AsString())

				msg, ok := m[logMessageKey]
				require.True(t, ok)
				require.Equal(t, "hello", msg.AsString())

				require.ElementsMatch(t, []zapcore.Field{zap.String("MyTraceId", "123"), zap.String("MySpanID", "456")}, logs.All()[0].Context)

				requireCodeAttrs(t, m)
			},
		},
	}

	for i, test := range tests {
		test := test
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			observedZapCore, observedLogs := observer.New(zap.InfoLevel)
			observedLogger := zap.New(observedZapCore)
			logger := New(observedLogger, WithMinLevel(zap.InfoLevel))
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
			tracer := provider.Tracer("test")

			ctx := context.Background()
			ctx, span := tracer.Start(ctx, "main")

			test.log(ctx, logger)

			span.End()

			spans := sr.Ended()
			require.Equal(t, 1, len(spans))

			events := spans[0].Events()
			require.Equal(t, 1, len(events))

			event := events[0]
			require.Equal(t, "log", event.Name)
			test.require(t, event, observedLogs)
		})
	}

	t.Run("providing extra fields to be recorded on the span, and logged", func(t *testing.T) {
		sr := tracetest.NewSpanRecorder()
		provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
		tracer := provider.Tracer("test")

		ctx := context.Background()
		ctx, span := tracer.Start(ctx, "main")

		core, observedLogs := observer.New(zap.InfoLevel)
		logger := New(zap.New(core), WithMinLevel(zap.InfoLevel))
		loggerWithCtx := logger.Ctx(ctx).Clone(WithExtraFields(
			zap.String("foo", "bar"),
			zap.String("MyTraceIDKey", span.SpanContext().TraceID().String()),
		))
		loggerWithCtx.Info("hello")

		span.End()

		spans := sr.Ended()
		require.Equal(t, 1, len(spans))

		events := spans[0].Events()
		require.Equal(t, 1, len(events))

		event := events[0]
		require.Equal(t, "log", event.Name)

		m := attrMap(event.Attributes)
		foo, ok := m["foo"]
		require.True(t, ok)
		require.Equal(t, "bar", foo.AsString())

		_, ok = m["MyTraceIDKey"]
		require.True(t, ok)
		requireCodeAttrs(t, m)

		require.Equal(t, 1, observedLogs.Len())
		require.Equal(t, "hello", observedLogs.All()[0].Message)
		require.Equal(t, zap.InfoLevel, observedLogs.All()[0].Level)

		contextMap := observedLogs.All()[0].ContextMap()
		require.Equal(t, "bar", contextMap["foo"])
		require.Equal(t, span.SpanContext().TraceID().String(), contextMap["MyTraceIDKey"])
	})
}

func requireCodeAttrs(t *testing.T, m map[attribute.Key]attribute.Value) {
	fn, ok := m[semconv.CodeFunctionKey]
	require.True(t, ok)
	require.Contains(t, fn.AsString(), "otelzap.TestOtelZap")

	file, ok := m[semconv.CodeFilepathKey]
	require.True(t, ok)
	require.Contains(t, file.AsString(), "otelzap/otelzap_test.go")

	_, ok = m[semconv.CodeLineNumberKey]
	require.True(t, ok)
}

func attrMap(attrs []attribute.KeyValue) map[attribute.Key]attribute.Value {
	m := make(map[attribute.Key]attribute.Value, len(attrs))
	for _, kv := range attrs {
		m[kv.Key] = kv.Value
	}
	return m
}
