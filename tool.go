package mcp

import (
	"reflect"
	"strings"
)

type McpToolDescriptor struct {
	Name        string             `json:"name"`        // Name of the tool
	Description string             `json:"description"` // Description of the tool
	InputSchema McpToolInputSchema `json:"inputSchema"` // Input schema of the tool
}

type McpToolInputSchema struct {
	Type       string                                `json:"type"`               // Always "object"
	Properties map[string]McpToolInputSchemaProperty `json:"properties"`         // Properties of the tool. Key is the property name
	Required   []string                              `json:"required,omitempty"` // Required properties of the tool
}

type McpToolInputSchemaProperty struct {
	Type        string `json:"type"`        // Type of the property
	Description string `json:"description"` // Description of the property
}

type McpToolOutput struct {
	Type     McpToolOutputType `json:"type"`               // Type of the output
	Text     string            `json:"text,omitempty"`     // Text of the output
	Data     string            `json:"data,omitempty"`     // Data of the output
	MimeType string            `json:"mimeType,omitempty"` // Mime type of the output
}

func (o McpToolOutput) String() string {
	switch o.Type {
	case McpToolOutputTypeText,
		McpToolOutputTypeError:
		return "\"" + o.Text + "\"" + " (" + string(o.Type) + ")"
	default:
		return o.MimeType + " (data)"
	}
}

type McpToolOutputType string

const McpToolOutputTypeText McpToolOutputType = "text"
const McpToolOutputTypeError McpToolOutputType = "error"

type McpTool struct {
	Name        string             // Name of the tool
	Description string             // Description of the tool
	Function    any                // Function to be called
	Parameters  []McpToolParameter // Properties of the tool
	Output      []McpToolParameter // Output of the tool
}

// String returns the name and parameters of the tool
// in the format "name (param1,param2) -> output1,output2"
func (t McpTool) String() string {
	params := make([]string, len(t.Parameters))
	for i, p := range t.Parameters {
		params[i] = p.String()
	}

	outs := make([]string, len(t.Output))
	for i, o := range t.Output {
		outs[i] = o.String()
	}

	return t.Name + " (" + strings.Join(params, ",") + ")" + " -> " + strings.Join(outs, ",")
}

type McpToolParameter struct {
	Name        string
	Description string
	Type        McpToolParameterType
	Kind        reflect.Kind
}

// String returns the name and type of the parameter
// in the format "name type"
// For example: "a string"
func (p McpToolParameter) String() string {
	return p.Name + " (" + string(p.Type) + ")"
}

type McpToolParameterType string

const McpToolParameterTypeString McpToolParameterType = "string"
const McpToolParameterTypeNumber McpToolParameterType = "number"
const McpToolParameterTypeBoolean McpToolParameterType = "boolean"
const McpToolParameterTypeError McpToolParameterType = "error"
const McpToolParameterTypeContext McpToolParameterType = "context"
