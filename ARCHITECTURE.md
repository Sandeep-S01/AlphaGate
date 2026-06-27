# Sentra Architecture

## Overview

Sentra is a modular, event-driven cryptocurrency trading platform foundation written in Go. The architecture emphasizes loose coupling between components through event-driven communication using Redis Streams, with PostgreSQL serving as the persistent store for durable state.

## Core Architectural Principles

### 1. Modularity
The system is divided into distinct modules that communicate through well-defined interfaces:
- **Exchange**: Handles connections to cryptocurrency exchanges (currently Binance)
- **Market Data**: Collects, processes, and stores market data candles
- **Strategy**: Evaluates trading signals based on market data
- **Risk**: Evaluates signals against risk parameters
- **Execution**: Executes trades (paper trading only in current implementation)
- **Database**: Manages PostgreSQL connections and queries
- **Monitoring**: Provides observability through metrics and logging
- **Orchestration**: Coordinates the workflow between modules

### 2. Event-Driven Communication
Modules communicate asynchronously through Redis Streams:
- Each module publishes events to specific streams
- Other modules consume from streams they're interested in
- This decouples modules in time and space
- Enables easy addition of new modules that react to existing events

### 3. Idempotency
All operations are designed to be idempotent to handle duplicate events safely:
- Pipeline runs use idempotency keys based on exchange, symbol, interval, and candle open time
- Database operations check for existing records before creating new ones
- Event processing skips already-processed events

### 4. Configuration-Driven Behavior
System behavior is controlled through environment variables and configuration:
- Feature flags enable/disable major components (market data collection, persistence, orchestration, etc.)
- Strategy parameters are configurable and stored in the database
- Risk parameters are configurable and stored in the database
- Connection details, ports, and timeouts are configurable

### 5. Observability
The system provides comprehensive observability:
- Structured logging with standardized fields
- Prometheus-compatible metrics endpoint
- Health check endpoints (/health, /ready)
- Operational endpoints for monitoring pipeline runs and stream states
- Distributed tracing through correlation ID propagation

## Component Details

### Market Data Module
- Collects klines/candles from exchanges via WebSocket connections
- Persists candles to PostgreSQL for historical analysis
- Publishes closed candle events to Redis Streams
- Supports backfilling historical data
- Includes redundancy and reconnection handling

### Strategy Module
- Evaluates trading signals based on historical candle data
- Supports multiple strategy types (SMA crossover, RSI mean reversion, BTC trend pullback, Pine Script custom)
- Strategies are configurable and stored in the database
- Plug-in architecture makes it easy to add new strategies
- Publishes signals to Redis Streams for consumption by risk module

### Risk Module
- Evaluates signals against configurable risk parameters
- Implements various risk controls (signal strength limits, position limits, daily loss limits, etc.)
- Supports both permissive and restrictive modes
- Publishes risk decisions to Redis Streams
- Maintains audit trail of risk decisions and rule evaluations

### Execution Module
- Simulates trade execution (paper trading only)
- Tracks virtual account balances and positions
- Implements realistic fee models and slippage
- Publishes execution events to Redis Streams
- Provides reset and manual cycle capabilities

### Orchestration Module (Worker)
- Coordinates the end-to-end trading workflow:
  1. Consumes closed candle events from market data stream
  2. Runs strategy evaluation on new candle data
  3. Evaluates resulting signals through risk management
  4. Executes approved decisions in paper trading mode
- Uses idempotency records to prevent duplicate processing
- Handles failures and retries gracefully
- Supports selective enabling of features via environment variables

### API Module
- Provides RESTful HTTP interface for system interaction
- Serves static dashboard assets
- Implements authentication (optional, API key based)
- Provides endpoints for:
  - Market data queries (candles, coverage, backfills)
  - Strategy configuration and evaluation
  - Risk configuration and decisions
  - Paper trading account management
  - System operations and monitoring
  - Audit and reporting
- Includes security headers and rate limiting

## Data Flow

### Normal Operation (Orchestration Enabled)
1. Market Data Collector receives candle data from exchange WebSocket
2. Collector persists candle to PostgreSQL
3. Collector publishes closed candle event to `stream:market-data`
4. Orchestration Worker consumes candle event
5. Worker retrieves lookback period of candles from database
6. Worker runs strategy evaluation on candle data
7. Worker publishes strategy signal to `stream:strategy-signals`
8. Risk module consumes signal and evaluates against risk rules
9. Risk module publishes decision to `stream:risk-decisions`
10. Execution module consumes approved decision and simulates trade
11. Execution module publishes trade result to `stream:execution-results`
12. Orchestration Worker records pipeline completion in database

### Manual Operation
Users can interact with the system through the API:
- Manual strategy evaluation via `/api/v1/strategy/evaluate`
- Manual risk evaluation via risk decision inspection
- Manual paper trading cycles via `/api/v1/paper/cycles`
- Backtesting and strategy comparison via dedicated endpoints
- System configuration through settings endpoints
- Account and position management through paper trading endpoints

## Extension Points

### Adding New Strategies
1. Implement the `strategy.Evaluator` interface
2. Add strategy type to constants in `types.go`
3. Add factory logic in `factory.go`
4. Add validation in `settings.go`
5. Update any required SQL queries in `settings.go`

### Adding New Risk Rules
1. Implement risk evaluation logic in the risk module
2. Add configuration to risk settings if needed
3. Update validation logic
4. Ensure proper audit logging

### Adding New Event Types
1. Define new event structure if needed
2. Update publishers to send events to appropriate streams
3. Update consumers to handle new event types
4. Add any necessary database persistence

### Adding New API Endpoints
1. Implement handler function in `internal/api/handler.go`
2. Add route to router in `internal/api/router.go`
3. Add any necessary middleware (authentication, validation, etc.)
4. Update API documentation in README.md

## Technology Choices

### Language: Go
- Chosen for performance, concurrency model, and ease of deployment
- Strong standard library reduces external dependencies
- Excellent support for backend services and CLI tools

### Database: PostgreSQL
- Selected for reliability, feature set, and operational maturity
- JSONB column type provides flexibility for evolving schemas
- Strong ACID guarantees ensure data consistency
- Wide tooling and community support

### Event Infrastructure: Redis Streams
- Provides durable, consumer-group-based event streaming
- Built-in consumer group support for scalable processing
- Persistent storage with configurable retention
- Rich querying capabilities

### Configuration: Environment Variables
- Follows twelve-factor app principles
- Easy to configure in different environments (local, CI, production)
- Clear separation between code and configuration
- Supports secret management integration

## Deployment Architecture

### Containerized Deployment
- Single Docker image contains all Go binaries
- Docker Compose orchestrates service dependencies
- Profile-based service enabling/disabling
- Environment-based configuration

### Production Considerations
- Authentication required in production mode
- Resource limits and timeouts configured appropriately
- Health checks for orchestration and monitoring
- Structured logging suitable for log aggregation systems
- Metrics endpoint for Prometheus integration
- Audit trail for compliance and debugging

## Security Considerations

### Authentication
- Optional API key authentication for protected endpoints
- Admin API key required when authentication is enabled
- Authentication can be disabled for development/testing

### Data Protection
- No persistent storage of API keys or secrets in code
- Secrets should be managed through environment variables or secret management systems
- Sensitive data is not logged in plaintext

### Network Security
- Services bind to configured interfaces only
- Docker networks limit inter-service communication
- Reverse proxy recommended for production SSL termination

### Input Validation
- All API inputs are validated
- Configuration values are validated at startup
- Database queries use parameterized statements to prevent injection

## Future Enhancements

The architecture is designed to support future enhancements:
- Live trading execution (with appropriate risk controls)
- Additional exchange integrations
- Advanced order types and execution algorithms
- Machine learning-based strategy evaluation
- Enhanced portfolio and risk analytics
- Multi-account and multi-strategy support
- Advanced notification and alerting systems