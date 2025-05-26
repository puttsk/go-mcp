package mcp

import (
	"encoding/base64"
	"reflect"
	"strings"
)

type McpToolDescriptor struct {
	Name        string             `json:"name"`        // Name of the tool
	Description string             `json:"description"` // Description of the tool
	InputSchema McpToolInputSchema `json:"inputSchema"` // Input schema of the tool
}

type McpToolInputSchema struct {
	Type        string                        `json:"type"`                  // Data type of tool input
	Description string                        `json:"description,omitempty"` // Description of the property
	Properties  map[string]McpToolInputSchema `json:"properties,omitempty"`  // Properties of the tool. Key is the property name
	Required    []string                      `json:"required,omitempty"`    // Required properties of the tool
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
	case McpToolOutputTypeImage:
		return o.MimeType + " (image)"
	default:
		return o.MimeType + " (data)"
	}
}

// McpToolOutputType represents the data type of the tool output.
type McpToolOutputType string

const McpToolOutputTypeText McpToolOutputType = "text"
const McpToolOutputTypeImage McpToolOutputType = "image"
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

// SetDescription sets the description of the tool
func (t *McpTool) SetDescription(description string) *McpTool {
	t.Description = description
	return t
}

// McpToolParameter represents tool parameters with go-compatible data type.
type McpToolParameter struct {
	Name        string
	Description string
	Type        McpToolDataType
	Kind        reflect.Kind
}

// String returns the name and type of the parameter
// in the format "name type"
// For example: "a string"
func (p McpToolParameter) String() string {
	return p.Name + " (" + string(p.Type) + ")"
}

type McpToolDataType string

const McpToolDataTypeString McpToolDataType = "string"
const McpToolDataTypeNumber McpToolDataType = "number"
const McpToolDataTypeBoolean McpToolDataType = "boolean"
const McpToolDataTypeError McpToolDataType = "error"
const McpToolDataTypeContext McpToolDataType = "context"
const McpToolDataTypeImage McpToolDataType = "image"

type McpImageMimeType string

const McpImageMimeTypePNG McpImageMimeType = "image/png"
const McpImageMimeTypeJPG McpImageMimeType = "image/jpg"
const McpImageMimeTypeJPEG McpImageMimeType = "image/jpeg"

type McpImage struct {
	MimeType McpImageMimeType `json:"mimeType"` // Mime type of the image
	Data     string           `json:"data"`     // Base64 encoded image data
}

func (img *McpImage) GetImageBinary() ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(img.Data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
