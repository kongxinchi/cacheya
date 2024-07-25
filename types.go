package cacheya

import "context"

type Marshaler interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

type Compressor interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

// Logger 日志接口
type Logger interface {
	Debug(ctx context.Context, message string, fields map[string]any)
	Info(ctx context.Context, message string, fields map[string]any)
	Warn(ctx context.Context, message string, fields map[string]any)
	Error(ctx context.Context, message string, fields map[string]any)
}

func DefaultNopLogger() Logger {
	return nopLogger{}
}

type nopLogger struct{}

func (l nopLogger) Debug(ctx context.Context, message string, fields map[string]any) {}
func (l nopLogger) Info(ctx context.Context, message string, fields map[string]any)  {}
func (l nopLogger) Warn(ctx context.Context, message string, fields map[string]any)  {}
func (l nopLogger) Error(ctx context.Context, message string, fields map[string]any) {}

// Metrics 监控埋点接口
type Metrics interface {
	Counter(ctx context.Context, name string, inc float64, labels map[string]string)
	Timer(ctx context.Context, name string, seconds float64, labels map[string]string)
	Value(ctx context.Context, name string, value float64, labels map[string]string)
}

func DefaultNopMetrics() Metrics {
	return nopMetrics{}
}

type nopMetrics struct{}

func (m nopMetrics) Counter(ctx context.Context, name string, inc float64, labels map[string]string) {
}
func (m nopMetrics) Timer(ctx context.Context, name string, seconds float64, labels map[string]string) {
}
func (m nopMetrics) Value(ctx context.Context, name string, value float64, labels map[string]string) {
}
