package tools

import (
	"context"
	"encoding/json"

	"github.com/giantswarm/mcp-toolkit/tooldesc"
	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

func registerListThings(s *mcpsrv.MCPServer, deps Deps) {
	tool := mcp.NewTool("things_list",
		tooldesc.Describe("List every thing the server knows about.", ""),
		readOnly(),
		mcp.WithOpenWorldHintAnnotation(false),
	)
	s.AddTool(tool, listThingsHandler(deps))
}

func listThingsHandler(deps Deps) mcpsrv.ToolHandlerFunc {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		things, err := deps.Client.List(ctx)
		if err != nil {
			return mcp.NewToolResultError("list things: " + err.Error()), nil
		}
		body, err := json.Marshal(things)
		if err != nil {
			return mcp.NewToolResultError("marshal things: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(body)), nil
	}
}
