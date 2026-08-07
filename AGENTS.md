# Repository Guidelines

## Project Structure & Module Organization
`main.go` starts the Gin API and wires dependencies through `wire.go` and generated `wire_gen.go`. Core backend layers follow `controller/ -> service/ -> repository/ -> domain/`. Versioned request and response DTOs live in `api/request/v1`, `api/request/v2`, and `api/response/...`. Cross-cutting code sits in `middleware/`, `ioc/`, and `pkg/`. Configuration examples are in `config/`, generated API docs are in `docs/`, and shared LLM logic is under `llm/`. The separate Python inference service lives in `llmservice/`.

## Build, Test, and Development Commands
Use `go run .` to start the API locally on `:8080`. The app expects `NACOSDSN` or a local `config/config.yaml` fallback.

- `make test`: run all Go tests with verbose output.
- `make generate`: refresh generated code from `go generate ./...` such as Wire and mocks.
- `make swag`: format Swagger annotations and regenerate `docs/swagger.yaml` and `docs/openapi3.yaml`.
- `make run_py`: start `llmservice` with `uv run main.py`.
- `docker build -t feedback:dev .`: build the Go service image used by `docker-compose.yaml`.

## Coding Style & Naming Conventions
Keep Go code `gofmt`-formatted; use standard Go naming: lowercase package names, `PascalCase` exports, `camelCase` internals. Keep HTTP handling in `controller/`, business rules in `service/`, and storage logic in `repository/`. Follow existing file names such as `sheet.go`, `chat.go`, and `auth.go`. In `llmservice/`, use 4-space indentation and `snake_case`. Do not hand-edit generated files like `wire_gen.go`, Swagger outputs in `docs/`, or mocks under `service/mock` and `pkg/*/mock`.

## Testing Guidelines
Tests use Go's `testing` package with `testify/assert` and `gomock`; `controller/sheet_test.go` is the current reference style. Add tests beside the package they cover and name them `*_test.go`. Prefer table-driven tests for handlers and services. No coverage gate is defined, so contributors are expected to add or extend tests for behavior changes.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commit prefixes such as `feat:` and `fix:`, sometimes with short Chinese summaries. Keep commits focused and include generated artifacts only when the source change requires regeneration. PRs should describe behavior changes, note config or schema impact, link the related issue, and include verification such as `make test` output or manual API checks. If an endpoint changes, regenerate Swagger/OpenAPI files in the same PR.

## Security & Configuration Tips
Use `config/example_config.yaml` as the template for local `config/config.yaml`. Never commit real Lark, JWT, Redis, MySQL, or Nacos credentials. `docker-compose.yaml` also expects `NACOSDSN` from the environment.
