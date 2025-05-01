package logger

import (
    "go.uber.org/zap"
    "sync"
)

var (
    once     sync.Once
    instance *zap.Logger
)

func GetLogger() *zap.Logger {
    once.Do(func() {
        cfg := zap.NewProductionConfig()
        cfg.OutputPaths = []string{"stdout"}
        logger, err := cfg.Build()
        if err != nil {
            panic(err)
        }
        instance = logger
    })
    return instance
}
