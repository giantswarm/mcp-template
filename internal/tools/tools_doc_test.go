package tools

import (
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/giantswarm/mcp-toolkit/tooldesc"
	"github.com/giantswarm/mcp-toolkit/tooldesc/tooldesctest"
	mcpsrv "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-template/internal/example"
)

// docsServer registers every tool; the descriptions need none of the
// dependencies.
var docsServer = sync.OnceValue(func() *mcpsrv.MCPServer {
	s := mcpsrv.NewMCPServer("docs", "docs")
	Register(s, Deps{Client: example.NewFakeClient(), Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	return s
})

// TestToolsDocIsCurrent keeps docs/tools.md — every tool's description and
// input schema as a model reads them — equal to the registered tools.
// TOOLS_DOC_UPDATE=1 rewrites the file (`make tools-doc`).
func TestToolsDocIsCurrent(t *testing.T) {
	tooldesctest.AssertServerDoc(t, docsServer(),
		"# The tools\n\nGenerated from the registered tools (`make tools-doc`); through muster every tool is `x_<server>_<name>`.",
		"../../docs/tools.md")
}

// TestToolDescriptionBudgets holds the registered descriptions to the
// budget of the mcp-toolkit tool-description conventions.
func TestToolDescriptionBudgets(t *testing.T) {
	if err := tooldesc.CheckServer(docsServer(), tooldesc.DefaultBudget); err != nil {
		t.Error(err)
	}
}

// TestReadOnlyToolsAreNotDestructive: a read-only tool says so in full, or
// mcp-go's defaults report it destructive and not idempotent.
func TestReadOnlyToolsAreNotDestructive(t *testing.T) {
	for _, st := range docsServer().ListTools() {
		a := st.Tool.Annotations
		if a.ReadOnlyHint == nil || !*a.ReadOnlyHint {
			continue
		}
		if a.DestructiveHint == nil || *a.DestructiveHint {
			t.Errorf("%s is read-only but reported destructive", st.Tool.Name)
		}
		if a.IdempotentHint == nil || !*a.IdempotentHint {
			t.Errorf("%s is read-only but not reported idempotent", st.Tool.Name)
		}
	}
}
