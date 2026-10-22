# Custom targets — extends the devctl-generated Makefile.gen.*.mk (which already
# provides build, test, lint, fmt, vet, install, run and the release packaging;
# the generated `build` is what stamps pkg/project, so it is not overridden here).

NAME ?= mcp-template
CHART_DIR ?= ./helm/$(NAME)

##@ Develop

.PHONY: run-stdio
run-stdio: ## Run on stdio (local dev / Claude Desktop)
	go run . serve --transport=stdio

.PHONY: run-http
run-http: ## Run on streamable-HTTP without OAuth (local dev only)
	OAUTH_ENABLED=false go run . serve --transport=streamable-http

##@ Testing

.PHONY: test-vet
test-vet: ## Run go test with coverage and go vet
	NO_COLOR=true go test -cover ./...
	go vet ./...

.PHONY: tools-doc
tools-doc: ## Regenerate docs/tools.md from the registered tools (the test TestToolsDocIsCurrent keeps it current)
	TOOLS_DOC_UPDATE=1 go test -count=1 -run TestToolsDocIsCurrent ./internal/tools

##@ Helm

.PHONY: helm-lint
helm-lint: ## helm lint the chart
	helm lint $(CHART_DIR)

.PHONY: helm-template
helm-template: ## Render chart with default values
	helm template $(NAME) $(CHART_DIR)

##@ Tidy

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy
