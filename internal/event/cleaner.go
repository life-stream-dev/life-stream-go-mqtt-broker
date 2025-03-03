package event

import (
	"context"
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Callable interface {
	Invoke(ctx context.Context) error
}

type Cleaner struct {
	cleaners       []Callable
	mu             sync.Mutex
	initOnce       sync.Once
	cleaning       bool
	loggerShutdown Callable
}

var cleanerInstance = &Cleaner{}

func NewCleaner() *Cleaner {
	return cleanerInstance
}

func (c *Cleaner) Add(callable Callable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cleaning {
		logger.Debug("Cleaner is already shutting down, ignoring new cleaner")
		return
	}
	c.cleaners = append(c.cleaners, callable)
}

func (c *Cleaner) Init(loggerShutdown Callable) {
	c.initOnce.Do(func() {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

		c.loggerShutdown = loggerShutdown

		go func() {
			<-ctx.Done()
			stop()
			logger.Info("Received interrupt signal, shutting down")

			c.mu.Lock()
			c.cleaning = true // 标记为清理中，阻止后续Add操作
			cleanersCopy := make([]Callable, len(c.cleaners))
			copy(cleanersCopy, c.cleaners)
			c.mu.Unlock()

			logger.DebugF("Starting cleanup of %d registered functions", len(cleanersCopy))

			var errs []error
			for i, callable := range cleanersCopy {
				func(idx int, c Callable) { // 使用匿名函数确保defer在每次迭代执行
					logger.DebugF("Invoking cleaner #%d (%T)", idx+1, c)
					timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancelFunc() // 确保每次调用后取消上下文
					if err := c.Invoke(timeoutCtx); err != nil {
						logger.ErrorF("Cleaner #%d (%T) failed: %v", idx+1, c, err) // 记录类型和错误
						errs = append(errs, err)
					}
				}(i, callable)
			}

			if len(errs) > 0 {
				logger.ErrorF("%d errors occurred during cleanup:", len(errs))
				for i, err := range errs {
					logger.ErrorF("Error %d: %v", i+1, err)
				}
			} else {
				logger.Debug("All cleaners executed successfully")
			}
			logger.Info("Cleanup finished, server offline")

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := c.loggerShutdown.Invoke(shutdownCtx); err != nil {
				fmt.Fprintf(os.Stderr, "LOGGER SHUTDOWN ERROR: %v\n", err)
			}
			syscall.Exit(0)
		}()
	})
}
