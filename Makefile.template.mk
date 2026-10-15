# The template's own helm-test. Sorts after Makefile.gen.*.mk, so this recipe is the
# one `make helm-test` runs, and so does the generated chart-tests workflow.
#
# The chart under helm/{MCP-NAME} is a template: its name renders into every
# metadata.name, which is no YAML until the repository is bootstrapped. Its suites run
# against the bootstrapped chart in the template-bootstrap-gate workflow.

.PHONY: helm-test
helm-test: ## Chart unit tests (run by the template-bootstrap-gate on the bootstrapped chart)
	@echo "====> $@: helm/{MCP-NAME} renders only once bootstrapped; template-bootstrap-gate runs its suites"
