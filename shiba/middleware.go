package shiba

import (
	"context"
	"io"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

type MiddlewareFunc func(http.Handler) http.Handler

func newJaegerTracer(serviceName, agentHostPort string) (io.Closer, error) {
	cfg := &config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  agentHostPort,
		},
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, nil
}

func MiddlewareTracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var newCtx context.Context
		var span opentracing.Span
		spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		if err != nil {
			span, newCtx = opentracing.StartSpanFromContextWithTracer(r.Context(), opentracing.GlobalTracer(), r.URL.Path)
		} else {
			span, newCtx = opentracing.StartSpanFromContextWithTracer(
				r.Context(),
				opentracing.GlobalTracer(),
				r.URL.Path,
				opentracing.ChildOf(spanCtx),
				opentracing.Tag{Key: string(ext.Component), Value: "HTTP"},
			)
		}
		defer span.Finish()

		var traceID string
		var spanID string
		var spanContext = span.Context()
		switch spanContext.(type) {
		case jaeger.SpanContext:
			jaegerContext := spanContext.(jaeger.SpanContext)
			traceID = jaegerContext.TraceID().String()
			spanID = jaegerContext.SpanID().String()
		}
		r.Header.Set("X-Trace-ID", traceID)
		r.Header.Set("X-Span-ID", spanID)
		r = r.WithContext(newCtx)

		next.ServeHTTP(w, r)
	})
}

func MiddlewareRecover(handler func(http.ResponseWriter, *http.Request, interface{})) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					defaultLogger.Errorf("---------- program crash: %+v ----------", err)
					defaultLogger.Errorf("---------- program crash: %s ----------", stack)

					if handler != nil {
						handler(w, r, err)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
