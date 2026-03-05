package mcpgen

// RenameTools applies a name mapping to the tools list.
// Keys in the mapping are original tool names, values are new names.
func RenameTools(tools []MCPTool, mapping map[string]string) []MCPTool {
	if len(mapping) == 0 {
		return tools
	}
	result := make([]MCPTool, len(tools))
	for i, tool := range tools {
		if newName, ok := mapping[tool.Name]; ok {
			tool.Name = newName
		}
		result[i] = tool
	}
	return result
}
