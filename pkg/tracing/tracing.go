package tracing

import (
	"context"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/maplerime/cl-query/pkg/common"
)

const (
	Rest       = "Rest"
	Middleware = "Middleware"
	Cache      = "Cache"
	Db         = "Db"
)

func NewId(ctx context.Context) context.Context {
	ctx = RealCtx(ctx)
	requestId, traceId, spanId := GenId(ctx)
	if traceId != "" {
		ctx = context.WithValue(ctx, common.MiddlewareTraceIdCtxKey, traceId)
		ctx = context.WithValue(ctx, common.MiddlewareSpanIdCtxKey, spanId)
	} else {
		ctx = context.WithValue(ctx, common.MiddlewareRequestIdCtxKey, requestId)
	}
	return ctx
}

func NewGinId(ctx context.Context) *gin.Context {
	keys := make(map[string]interface{})
	ctx = RealCtx(ctx)
	requestId, traceId, spanId := GenId(ctx)
	if traceId != "" {
		keys[common.MiddlewareTraceIdCtxKey] = traceId
		keys[common.MiddlewareSpanIdCtxKey] = spanId
	} else {
		keys[common.MiddlewareRequestIdCtxKey] = requestId
	}
	return &gin.Context{
		Keys: keys,
	}
}

func GenId(ctx context.Context) (string, string, string) {
	ctx = RealCtx(ctx)
	requestId := RequestId(ctx)
	traceId, spanId := TraceId(ctx)
	if traceId != "" {
		requestId = traceId
	}
	// gen uuid
	if requestId == "" {
		requestId = uuid.NewString()
	}
	return requestId, traceId, spanId
}

func GetId(ctx context.Context) (string, string, string) {
	ctx = RealCtx(ctx)
	requestId := RequestId(ctx)
	traceId, spanId := TraceId(ctx)
	if traceId != "" {
		requestId = traceId
	}
	return requestId, traceId, spanId
}

func RequestId(ctx context.Context) (id string) {
	ctx = RealCtx(ctx)
	// get value from context
	requestIdValue := ctx.Value(common.MiddlewareRequestIdCtxKey)
	if item, ok := requestIdValue.(string); ok && item != "" {
		id = item
	}
	return
}

func TraceId(ctx context.Context) (traceId, spanId string) {
	ctx = RealCtx(ctx)
	span := trace.SpanFromContext(ctx).SpanContext()
	if span.IsValid() {
		traceId = span.TraceID().String()
		spanId = span.SpanID().String()
	}
	return
}

func RealCtx(ctx context.Context) context.Context {
	if interfaceIsNil(ctx) {
		return context.Background()
	}
	if c, ok := ctx.(*gin.Context); ok {
		// gin context contains cancel ctx, remove it
		ctx = c.Request.Context()
		requestId, traceId, spanId := GetId(ctx)
		if traceId != "" {
			ctx = context.WithValue(ctx, common.MiddlewareTraceIdCtxKey, traceId)
			ctx = context.WithValue(ctx, common.MiddlewareSpanIdCtxKey, spanId)
		} else {
			ctx = context.WithValue(ctx, common.MiddlewareRequestIdCtxKey, requestId)
		}
	}
	return ctx
}

func Name(name ...string) string {
	return strings.Join(name, ".")
}

func interfaceIsNil(i interface{}) bool {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		return v.IsNil()
	}
	return i == nil
}
