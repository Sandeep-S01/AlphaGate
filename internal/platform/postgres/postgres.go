package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sentra/internal/config"
	"sentra/internal/observability"
)

func Connect(ctx context.Context, cfg config.PostgresConfig, metrics *observability.Registry) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	// Set connection pool parameters from config
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)

	if cfg.MaxConnLifetime != "" {
		maxConnLifetime, err := time.ParseDuration(cfg.MaxConnLifetime)
		if err != nil {
			return nil, fmt.Errorf("invalid postgres max_conn_lifetime: %w", err)
		}
		poolConfig.MaxConnLifetime = maxConnLifetime
	} else {
		poolConfig.MaxConnLifetime = time.Hour
	}

	if cfg.MaxConnIdleTime != "" {
		maxConnIdleTime, err := time.ParseDuration(cfg.MaxConnIdleTime)
		if err != nil {
			return nil, fmt.Errorf("invalid postgres max_conn_idle_time: %w", err)
		}
		poolConfig.MaxConnIdleTime = maxConnIdleTime
	} else {
		poolConfig.MaxConnIdleTime = 30 * time.Minute
	}

	if cfg.HealthCheckPeriod != "" {
		healthCheckPeriod, err := time.ParseDuration(cfg.HealthCheckPeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid postgres health_check_period: %w", err)
		}
		poolConfig.HealthCheckPeriod = healthCheckPeriod
	} else {
		poolConfig.HealthCheckPeriod = time.Minute
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	// Start metrics reporting goroutine if metrics registry is provided
	if metrics != nil {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					stats := pool.Stat()
					if metrics != nil {
						metrics.SetDBOpenConnections(uint64(stats.TotalConns()))
						metrics.SetDBInUseConnections(uint64(stats.AcquiredConns()))
						metrics.SetDBIdleConnections(uint64(stats.IdleConns()))
						metrics.AddDBWaitCount(uint64(stats.EmptyAcquireCount()))
						metrics.AddDBWaitDuration(uint64(stats.AcquireDuration() / time.Millisecond))
					}
				}
			}
		}()
	}

	return pool, nil
}
