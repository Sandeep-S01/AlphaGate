# Contributing to Sentra

Thank you for your interest in contributing to Sentra! This document provides guidelines and instructions for contributing to the project.

## Development Setup

### Prerequisites

- Go 1.23+
- Docker and Docker Compose
- Git

### Local Development

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/yourusername/sentra.git
   cd sentra
   ```

2. Copy the example environment file and adjust as needed:
   ```powershell
   copy .env.example .env
   ```

3. Start the required dependencies (PostgreSQL and Redis):
   ```powershell
   docker compose up -d postgres redis
   ```

4. Install Go dependencies:
   ```bash
   go mod download
   ```

5. Run the test suite to verify your setup:
   ```bash
   go test ./...
   ```

6. Apply database migrations:
   ```powershell
   go run ./cmd/migrate
   ```

7. Start the API server:
   ```powershell
   go run ./cmd/api
   ```

8. In another terminal, start the worker:
   ```powershell
   go run ./cmd/worker
   ```

### Making Changes

1. Create a new branch for your feature or bug fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes, following the code style and conventions in the existing codebase.

3. Add or update tests as appropriate.

4. Ensure all tests pass:
   ```bash
   go test ./...
   ```

5. Commit your changes:
   ```bash
   git commit -m "Description of your changes"
   ```

6. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

7. Open a pull request against the main repository.

### Code Style

- Follow the existing Go code style in the repository.
- Use `gofmt` to format your code before committing.
- Write clear, descriptive commit messages.
- Add comments for complex logic or non-obvious implementations.
- Ensure all new code is tested.

### Running Tests

- Run all tests: `go test ./...`
- Run tests with coverage: `go test ./... -cover`
- Run tests for a specific package: `go test ./internal/strategy`
- Run tests with verbose output: `go test ./... -v`

### Database Migrations

- Migration files are located in the `migrations/` directory.
- To create a new migration:
  1. Create a new file with the format `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`
  2. The migration runner will automatically apply new migrations when starting
  3. Migrations are tracked in the `schema_migrations` table

### Adding New Strategies

Sentra is designed to be extensible with new trading strategies. To add a new strategy:

1. Implement the `strategy.Evaluator` interface in a new file under `internal/strategy/`
2. Add your strategy name to the `StrategyName` constants in `internal/strategy/types.go`
3. Update the `NewEvaluatorFromSettings` function in `internal/strategy/factory.go` to handle your new strategy
4. Add validation logic in the `Settings.Validate()` method in `internal/strategy/settings.go`
5. Add any required configuration fields to the `Settings` struct if needed
6. Update the `RequiredCandles()` method to return the minimum number of candles needed for your strategy

See the existing strategies (`sma.go`, `rsi.go`, `trend_pullback.go`) for examples of how to implement the Evaluator interface.

### Documentation

- Update any relevant documentation in the `docs/` directory
- Ensure API endpoints are documented if you add or modify them
- Keep the README.md up to date with any significant changes

## Reporting Issues

Please use the GitHub issue tracker to report bugs or request features. When reporting a bug, include:

- Steps to reproduce the issue
- Expected behavior vs. actual behavior
- Any relevant logs or error messages
- Information about your environment (Go version, Docker version, etc.)

## License

By contributing to Sentra, you agree that your contributions will be licensed under the project's license.