package cacheya

import "context"

type Marshaller interface {
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

func NewNoopLogger() Logger {
	return &noopLogger{}
}

type noopLogger struct{}

func (l *noopLogger) Debug(ctx context.Context, message string, fields map[string]any) {}
func (l *noopLogger) Info(ctx context.Context, message string, fields map[string]any)  {}
func (l *noopLogger) Warn(ctx context.Context, message string, fields map[string]any)  {}
func (l *noopLogger) Error(ctx context.Context, message string, fields map[string]any) {}

// Metrics 监控埋点接口
type Metrics interface {
	Counter(ctx context.Context, name string, inc float64, labels map[string]string)
	Timer(ctx context.Context, name string, seconds float64, labels map[string]string)
	Value(ctx context.Context, name string, value float64, labels map[string]string)
}

func NewNoopMetrics() Metrics {
	return &noopMetrics{}
}

type noopMetrics struct{}

func (m *noopMetrics) Counter(ctx context.Context, name string, inc float64, labels map[string]string) {
}
func (m *noopMetrics) Timer(ctx context.Context, name string, seconds float64, labels map[string]string) {
}
func (m *noopMetrics) Value(ctx context.Context, name string, value float64, labels map[string]string) {
}
