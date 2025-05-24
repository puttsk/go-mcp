package mcp

type McpInitializeResponse struct {
	ProtocolVersion McpProtocolVersion    `json:"protocolVersion"` // Protocol version
	Capabilities    McpServerCapabilities `json:"capabilities"`    // Capabilities
	ServerInfo      McpServerInfo         `json:"serverInfo"`      // Server info
	Instructions    string                `json:"instructions"`    // Instructions
}

type McpServerInfo struct {
	Name    string `json:"name"`    // Server name
	Version string `json:"version"` // Server version
}

type McpServerCapabilities struct {
	Logging   any                     `json:"logging,omitempty"`   // Logging capabilities
	Prompts   *McpCapabilityPrompts   `json:"prompts,omitempty"`   // Prompts capabilities
	Resources *McpCapabilityResources `json:"resources,omitempty"` // Resources capabilities
	Tools     *McpCapabilityTools     `json:"tools,omitempty"`     // Tools capabilities
}

type McpCapabilityPrompts struct {
	ListChanged bool `json:"listChanged"` // List changed
}
type McpCapabilityResources struct {
	ListChanged bool `json:"listChanged"` // List changed
	Subscribe   bool `json:"subscribe"`   // Subscribe
}
type McpCapabilityTools struct {
	ListChanged bool `json:"listChanged"` // List changed
}
