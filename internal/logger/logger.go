package logger

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

type Logger struct {
	*zap.Logger
}

func newLogger() *Logger {
	customLog, err := zap.NewProduction()
	if err != nil {
		log.Println(err)
	}
	return &Logger{Logger: customLog}
}

func CreateLogger(level string) (customLog *Logger, err error) {
	log := newLogger()
	defer log.Sync()

	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return log, err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return log, err
	}

	log.Logger = zl
	return log, nil
}

func (log *Logger) WithLogging() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()
			defer func() {
				log.Info("served",
					zap.String("method", r.Method),
					zap.String("uri", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Duration("duration", time.Since(t1)),
					zap.Int("size", ww.BytesWritten()))
			}()
			h.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
