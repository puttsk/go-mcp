package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
)

type McpProtocolVersion string

const McpProtocol2025_30_26 McpProtocolVersion = "2025-03-26" // MCP protocol version 2025-03-26

type McpMethodFunc func(ctx context.Context, req *McpRequest) (*McpResponse, error)

type JsonRPCVersion string

const JsonRPCVersion2_0 JsonRPCVersion = "2.0" // JSON-RPC version 2.0

type LogLevel int

const (
	LogLevelDebug LogLevel = iota // Debug log level
	LogLevelInfo                  // Info log level
)

type McpServer struct {
	LogLevel LogLevel // Log level

	TransportHandler McpTransportHandler // Transport handler
	SessionManager   McpSessionManager   // Session manager

	JsonRPC         JsonRPCVersion     // JSON-RPC version
	ProtocolVersion McpProtocolVersion // Protocol version

	Name         string // Server name
	Version      string // Server version
	Instructions string // Instructions describing how to use the server and its features.

	Methods map[string]McpMethodFunc // List of methods

	Logging   bool               // Enable logging
	Prompts   []any              // List of prompts
	Resources []any              // List of resources
	Tools     map[string]McpTool // List of tools
}

func NewMcpServer(name string, version string, protocolVersion McpProtocolVersion) (*McpServer, error) {
	switch protocolVersion {
	case McpProtocol2025_30_26:
		// Valid protocol version
	default:
		return nil, fmt.Errorf("unsupported protocol version: %s", protocolVersion)
	}

	s := &McpServer{
		LogLevel:        LogLevelInfo,
		JsonRPC:         JsonRPCVersion2_0,
		Name:            name,
		Version:         version,
		ProtocolVersion: protocolVersion,
		Methods:         make(map[string]McpMethodFunc),
		Logging:         false,
		Prompts:         []any{},
		Resources:       []any{},
		Tools:           make(map[string]McpTool),
	}

	// Register default methods
	s.RegisterMethod("initialize", s.MethodInitialize)
	s.RegisterMethod("notifications/initialized", s.MethodNotificationInitialized)
	s.RegisterMethod("tools/list", s.MethodToolsList)
	s.RegisterMethod("tools/call", s.MethodToolsCall)

	s.Logf("MCP Server initialized: %s v%s", s.Name, s.Version)

	return s, nil
}

// Debugf logs a debug message if the server is in debug mode.
func (s *McpServer) Debugf(format string, a ...interface{}) {
	if s.LogLevel <= LogLevelDebug {
		log.Printf(`[DEBUG] `+format, a...)
	}
}

// Logf logs a message.
func (s *McpServer) Logf(format string, a ...interface{}) {
	if s.LogLevel <= LogLevelInfo {
		log.Printf(`[ INFO] `+format, a...)
	}
}

// RegisterMethod registers a method with the server.
// If the method is already registered, it returns an error.
// The method name must be unique.
func (s *McpServer) RegisterMethod(name string, method McpMethodFunc) error {
	if s.Methods == nil {
		s.Methods = make(map[string]McpMethodFunc)
	}

	if _, exists := s.Methods[name]; exists {
		s.Logf("Method %s already registered", name)
		return fmt.Errorf("method %s already registered", name)
	}

	s.Methods[name] = method

	return nil
}

// CreateMcpResponse creates an MCP response with the given result
func (s *McpServer) CreateMcpResponse(ctx context.Context, result any) (*McpResponse, error) {
	reqID, err := GetRequestIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return &McpResponse{
		JsonRPC: s.JsonRPC,
		ID:      json.Number(reqID),
		Results: result,
	}, nil
}

// CreateMcpErrorResponse creates an MCP error response with the given error
func (s *McpServer) CreateMcpErrorResponse(ctx context.Context, mcpErr *McpError) (*McpResponse, error) {
	reqID, err := GetRequestIDFromContext(ctx)
	if err != nil {
		reqID = "-1" // Default request ID if not found in context
	}

	return &McpResponse{
		JsonRPC: s.JsonRPC,
		ID:      json.Number(reqID),
		Error:   mcpErr,
	}, nil
}

// ProcessRequest handles the incoming request and returns a response formatted for the transport layer.
// If the error originates from the MCP service, include the error details in the response and return a nil error.
// For internal server errors, return a non-nil error and a nil response.
func (s *McpServer) ProcessRequest(ctx context.Context, req any) (any, error) {
	var mcpSession McpSession

	if s.SessionManager == nil {
		return nil, fmt.Errorf("session manager is not set")
	}
	if s.TransportHandler == nil {
		return nil, fmt.Errorf("transport handler is not set")
	}

	// Transfrom request from transport layer (e.g. AWS Lambda with steamable HTTP) to MCP request
	mcpReq, err := s.TransportHandler.ProcessRequest(ctx, req)
	if err != nil {
		resp, _ := s.CreateMcpErrorResponse(ctx, NewMcpError(ErrInvalidRequestCode, "invalid request", nil))
		return s.TransportHandler.ProcessResponse(ctx, resp)
	}
	// Set request ID in context
	ctx = SetRequestIDInContext(ctx, mcpReq.ID.String())

	// Prepare session from the incoming request
	// If session is not set, create a new session
	sid, err := s.TransportHandler.GetSessionID(ctx, req)
	if err != nil && !errors.Is(err, ErrNoSessionHeader) {
		resp, _ := s.CreateMcpErrorResponse(ctx, NewErrInternalError("cannot get session", nil))
		return s.TransportHandler.ProcessResponse(ctx, resp)
	}

	if sid == "" {
		// Create a new session
		mcpSession, err = s.SessionManager.CreateSession()
		if err != nil {
			resp, _ := s.CreateMcpErrorResponse(ctx, NewErrInternalError("cannot create session", nil))
			return s.TransportHandler.ProcessResponse(ctx, resp)
		}
	} else {
		// Check if the session exists
		sess, ok := s.SessionManager.GetSession(sid)
		if !ok {
			resp, _ := s.CreateMcpErrorResponse(ctx, NewErrInternalError("cannot get session", nil))
			return s.TransportHandler.ProcessResponse(ctx, resp)
		}
		mcpSession = sess
	}

	// Setup session in context
	ctx = SetSessionInContext(ctx, mcpSession)

	// Process the request
	if _, ok := s.Methods[mcpReq.Method]; !ok {
		s.Logf("Method %s not found", mcpReq.Method)
		resp, _ := s.CreateMcpErrorResponse(ctx, NewErrUnknownMethod(mcpReq.Method))
		return s.TransportHandler.ProcessResponse(ctx, resp)
	}

	s.Logf("Processing method: %s", mcpReq.Method)
	resp, err := s.Methods[mcpReq.Method](ctx, mcpReq)
	if err != nil {
		s.Logf("Error processing method %s: %v", mcpReq.Method, err)
		resp, _ := s.CreateMcpErrorResponse(ctx, NewErrInternalError(err.Error(), nil))
		return s.TransportHandler.ProcessResponse(ctx, resp)
	}

	return s.TransportHandler.ProcessResponse(ctx, resp)
}

// MethodInitialize performs MCP initialize method and returns an MCP initialize response containing server information and capabilities.
// See. https://modelcontextprotocol.io/specification/2025-03-26/basic/lifecycle#initialization
func (s *McpServer) MethodInitialize(ctx context.Context, req *McpRequest) (*McpResponse, error) {
	sess, err := GetSessionFromContext(ctx)
	if err != nil {
		// Something went wrong with the session
		return nil, err
	}

	if sess.Initialized {
		return s.CreateMcpErrorResponse(ctx, ErrSessionAlreadyInitialized)
	}

	init := McpInitializeResponse{
		ProtocolVersion: s.ProtocolVersion,
		Capabilities:    McpServerCapabilities{},
		ServerInfo: McpServerInfo{
			Name:    s.Name,
			Version: s.Version,
		},
	}

	// Setup capabilities
	if s.Logging {
		init.Capabilities.Logging = map[string]any{}
	}

	if len(s.Prompts) > 0 {
		init.Capabilities.Prompts = &McpCapabilityPrompts{
			ListChanged: false,
		}
	}
	if len(s.Resources) > 0 {
		init.Capabilities.Resources = &McpCapabilityResources{
			ListChanged: false,
			Subscribe:   false,
		}
	}
	if len(s.Tools) > 0 {
		init.Capabilities.Tools = &McpCapabilityTools{
			ListChanged: false,
		}
	}

	init.Instructions = s.Instructions

	return s.CreateMcpResponse(ctx, init)
}

// MethodNotificationInitialized process MCP notification initialized message from the client to complete the hand shake process.
// See. https://modelcontextprotocol.io/specification/2025-03-26/basic/lifecycle#initialization
func (s *McpServer) MethodNotificationInitialized(ctx context.Context, req *McpRequest) (*McpResponse, error) {
	sess, err := GetSessionFromContext(ctx)
	if err != nil {
		// Somwthing went wrong with the session
		return nil, err
	}

	// Session already initialized
	if sess.Initialized {
		return s.CreateMcpErrorResponse(ctx, ErrSessionAlreadyInitialized)
	}

	// Set session as initialized
	_, err = s.SessionManager.SetSessionInitialized(sess, true)
	if err != nil {
		return nil, err
	}

	return s.CreateMcpResponse(ctx, nil)

}

// MethodToolsList process MCP tools/list method and returns a list of registered tools.
func (s *McpServer) MethodToolsList(ctx context.Context, req *McpRequest) (*McpResponse, error) {
	sess, err := GetSessionFromContext(ctx)
	if err != nil {
		// Somwthing went wrong with the session
		return nil, err
	}

	// Session is not initialized
	if !sess.Initialized {
		return s.CreateMcpErrorResponse(ctx, ErrSessionNotInitialized)
	}

	tools, err := s.ListTools()
	if err != nil {
		return nil, err
	}
	resp := map[string]any{"tools": tools}

	return s.CreateMcpResponse(ctx, resp)
}

// MethodToolsCall process MCP tools/call method and calls the specified tool with the given arguments.
func (s *McpServer) MethodToolsCall(ctx context.Context, req *McpRequest) (*McpResponse, error) {
	sess, err := GetSessionFromContext(ctx)
	if err != nil {
		// Somwthing went wrong with the session
		return nil, err
	}

	// Session is not initialized
	if !sess.Initialized {
		return s.CreateMcpErrorResponse(ctx, ErrSessionNotInitialized)
	}

	params, ok := req.Params.(map[string]any)
	if !ok {
		return s.CreateMcpErrorResponse(ctx, ErrInvalidMcpRequestParameters)
	}

	toolName := ""
	toolArgs := []any{}

	n, ok := params["name"]
	if !ok {
		return s.CreateMcpErrorResponse(ctx, NewMcpError(ErrInvalidParametersCode, "missing tool name", nil))
	}
	toolName, ok = n.(string)
	if !ok {
		return s.CreateMcpErrorResponse(ctx, NewMcpError(ErrInvalidParametersCode, "invalid tool name", nil))
	}

	tool, ok := s.Tools[toolName]
	if !ok {
		return s.CreateMcpErrorResponse(ctx, NewMcpError(ErrMethodNotFoundCode, "tool not found", nil))
	}

	var callArgs map[string]any
	if a, ok := params["arguments"]; ok {
		args, ok := a.(map[string]any)
		if !ok {
			return s.CreateMcpErrorResponse(ctx, NewMcpError(ErrInvalidRequestCode, "arguments is not object", nil))
		}
		callArgs = args
	}

	s.Logf("[%s] Calling tool %s with arguments: %v", sess.SessionID, toolName, callArgs)

	// Convert MCP tool parameters to function arguments
	for _, p := range tool.Parameters {
		if p.Type == McpToolParameterTypeContext {
			// Slip for context.Context parameter
			continue
		}

		if _, ok := callArgs[p.Name]; !ok {
			s.Logf("Missing argument: %s", p.Name)
			return s.CreateMcpErrorResponse(ctx, NewMcpError(ErrInvalidParametersCode, "missing arguments", map[string]any{"argument": p.Name}))
		}
		val := callArgs[p.Name]
		switch p.Kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			num, ok := val.(json.Number)
			if !ok {
				s.Logf("Argument is not %s (actual: %s)", p.Kind, reflect.TypeOf(val))
				return s.CreateMcpErrorResponse(ctx, NewErrInvalidArgumentType(p.Name, p.Type))
			}
			param, err := num.Int64()
			if err != nil {
				s.Logf("Cannot cast argument to int64: %s", err)
				return s.CreateMcpErrorResponse(ctx, NewErrInvalidArgumentType(p.Name, p.Type))
			}
			switch p.Kind {
			case reflect.Int:
				toolArgs = append(toolArgs, int(param))
			case reflect.Int8:
				toolArgs = append(toolArgs, int8(param))
			case reflect.Int16:
				toolArgs = append(toolArgs, int16(param))
			case reflect.Int32:
				toolArgs = append(toolArgs, int32(param))
			case reflect.Uint:
				toolArgs = append(toolArgs, uint(param))
			case reflect.Uint8:
				toolArgs = append(toolArgs, uint8(param))
			case reflect.Uint16:
				toolArgs = append(toolArgs, uint16(param))
			case reflect.Uint32:
				toolArgs = append(toolArgs, uint32(param))
			case reflect.Uint64:
				toolArgs = append(toolArgs, uint64(param))
			default:
				toolArgs = append(toolArgs, param)
			}

		case reflect.Float32, reflect.Float64:
			num, ok := val.(json.Number)
			if !ok {
				s.Logf("argument is not %s (actual: %s)", p.Kind, reflect.TypeOf(val))
				return s.CreateMcpErrorResponse(ctx, NewErrInvalidArgumentType(p.Name, p.Type))
			}
			param, err := num.Float64()
			if err != nil {
				s.Logf("Cannot cast argument to float64: %s", err)
				return s.CreateMcpErrorResponse(ctx, NewErrInvalidArgumentType(p.Name, p.Type))
			}
			switch p.Kind {
			case reflect.Float32:
				toolArgs = append(toolArgs, float32(param))
			default:
				toolArgs = append(toolArgs, param)
			}
		case reflect.String:
			param, ok := val.(string)
			if !ok {
				s.Logf("argument is not %s (actual: %s)", p.Kind, reflect.TypeOf(val))
				return s.CreateMcpErrorResponse(ctx, NewErrInvalidArgumentType(p.Name, p.Type))
			}
			toolArgs = append(toolArgs, param)
		case reflect.Bool:
			param, ok := val.(bool)
			if !ok {
				s.Logf("argument is not %s (actual: %s)", p.Kind, reflect.TypeOf(val))
				return s.CreateMcpErrorResponse(ctx, NewErrInvalidArgumentType(p.Name, p.Type))
			}
			toolArgs = append(toolArgs, param)
		default:
			return s.CreateMcpErrorResponse(ctx, NewErrInvalidArgumentType(p.Name, p.Type))
		}
	}

	result, err := s.CallTool(ctx, toolName, toolArgs...)
	if err != nil {
		return nil, NewMcpError(ErrInternalErrorCode, "error calling tool", nil)
	}

	// Check if the tool returned an error
	for _, o := range result {
		if o.Type == McpToolOutputTypeError {
			resp := map[string]any{"content": []McpToolOutput{
				{
					Type: McpToolOutputTypeText,
					Text: o.Text,
				},
			}, "isError": true}
			return s.CreateMcpResponse(ctx, resp)
		}
	}

	resp := map[string]any{"content": result, "isError": false}
	return s.CreateMcpResponse(ctx, resp)
}

// Tool functions:

// ListTools returns a list of registered tools with their input schema.
func (s *McpServer) ListTools() ([]McpToolDescriptor, error) {
	if s.Tools == nil || len(s.Tools) == 0 {
		return nil, fmt.Errorf("no registered tools")
	}

	tools := make([]McpToolDescriptor, 0, len(s.Tools))
	for _, tool := range s.Tools {
		t := McpToolDescriptor{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: McpToolInputSchema{
				Type:       "object",
				Properties: make(map[string]McpToolInputSchemaProperty),
				Required:   []string{},
			},
		}
		for i, param := range tool.Parameters {

			// Do not expose context.Context as a parameter
			if param.Type == McpToolParameterTypeContext {
				if i != 0 {
					return nil, fmt.Errorf("context.Context must be the first parameter")
				}
				continue
			}

			t.InputSchema.Properties[param.Name] = McpToolInputSchemaProperty{
				Type:        string(param.Type),
				Description: param.Description,
			}

			// Make all parameters required since Go functions do not support optional parameters
			t.InputSchema.Required = append(t.InputSchema.Required, param.Name)
		}
		tools = append(tools, t)
	}

	return tools, nil
}

// RegisterTool registers a tool with the server.
func (s *McpServer) RegisterTool(name string, description string, tool any, params ...McpToolParameter) error {
	if s.Tools == nil {
		s.Tools = map[string]McpTool{}
	}

	if _, ok := s.Tools[name]; ok {
		return fmt.Errorf("tool %s already registered", name)
	}

	t := McpTool{
		Name:        name,
		Description: description,
		Function:    tool,
		Parameters:  []McpToolParameter{},
		Output:      []McpToolParameter{},
	}

	toolInfo := reflect.TypeOf(tool)
	contextOffset := 0

	// Check validity of the tool
	if toolInfo.Kind() != reflect.Func {
		return fmt.Errorf("tool must be a function")
	}

	// Check if the number of parameters matches the function signature
	// If the first parameter is context.Context, adjust the parameter offset accordingly
	if toolInfo.NumIn() > 0 {
		if toolInfo.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			contextOffset = 1
		}
	}

	// Check if the number of parameters matches the function signature
	if toolInfo.NumIn() != (len(params) + contextOffset) {
		return fmt.Errorf("parameter count does not match the number of tool parameters")
	}

	// Setup tool parameters
	for i := range toolInfo.NumIn() {
		arg := toolInfo.In(i)

		p := McpToolParameter{}

		// If the parameter is not context.Context, use the provided parameter name and description
		if contextOffset <= i {
			p.Name = params[i-contextOffset].Name
			p.Description = params[i-contextOffset].Description
			p.Kind = arg.Kind()
		}

		// Determine the type of the parameter based on its kind
		switch arg.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			// Number type
			p.Type = McpToolParameterTypeNumber
		case reflect.String:
			// String type
			p.Type = McpToolParameterTypeString
		case reflect.Bool:
			// Boolean type
			p.Type = McpToolParameterTypeBoolean
		case reflect.Interface:
			// Interface type

			// If the parameter is context.Context, set the type accordingly
			// Else return erros as unsupported type
			if arg.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
				p.Type = McpToolParameterTypeContext
				if i != 0 {
					return fmt.Errorf("context.Context must be the first parameter")
				}
			} else {
				return fmt.Errorf("unsupported function return type: %s", arg.Kind())
			}
		default:
			// Unsupported type
			return fmt.Errorf("unsupported function parameter type: %s", arg.Kind())
		}
		t.Parameters = append(t.Parameters, p)
	}

	// Setup tool output
	for i := range toolInfo.NumOut() {
		out := toolInfo.Out(i)

		// Set default output parameter name and type
		p := McpToolParameter{
			Name: fmt.Sprintf("out%d", i),
			Kind: out.Kind(),
		}

		// Setup output parameter type based on its kind
		switch out.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			// Number type
			p.Type = McpToolParameterTypeNumber
		case reflect.String:
			// String type
			p.Type = McpToolParameterTypeString
		case reflect.Bool:
			// Boolean type
			p.Type = McpToolParameterTypeBoolean
		case reflect.Interface:
			// Interface type
			if out.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				// Error type
				p.Type = McpToolParameterTypeError
			} else {
				return fmt.Errorf("unsupported function return type: %s", out.Kind())
			}
		default:
			// Unsupported type
			return fmt.Errorf("unsupported function return type: %s", out.Kind())
		}
		t.Output = append(t.Output, p)
	}

	s.Tools[name] = t

	s.Logf("Tool registered: %v", t)

	return nil
}

// CallTool calls a registered tool with the given name and parameters.
func (s *McpServer) CallTool(ctx context.Context, name string, params ...any) ([]McpToolOutput, error) {
	s.Logf("Tool %s called with args: %v", name, params)

	// Check if the tool is registered
	tool, ok := s.Tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	if tool.Function == nil {
		return nil, fmt.Errorf("tool %s has no function", name)
	}

	// Check if the registered function is a valid function
	f := reflect.ValueOf(tool.Function)
	if f.Kind() != reflect.Func {
		return nil, fmt.Errorf("tool %s is not a function", name)
	}

	// Build the arguments for the function call
	args := make([]reflect.Value, 0, len(tool.Parameters))
	contextOffset := 0

	// Setup context as first argument if the tool has a context parameter
	if len(tool.Parameters) > 0 {
		if tool.Parameters[0].Type == McpToolParameterTypeContext {
			contextOffset = 1
			// Add context as first argument
			args = append(args, reflect.ValueOf(ctx))
		}
	}

	// Setup the rest of the arguments based on the tool parameters
	for i, p := range tool.Parameters {
		if contextOffset <= i {
			arg := reflect.ValueOf(params[i-contextOffset])
			if arg.Kind() != p.Kind {
				return nil, fmt.Errorf("tool %s parameter %d has wrong type: expected %s, got %s", name, i, p.Kind, arg.Kind())
			}
			args = append(args, arg)
		}
	}

	s.Debugf("Tool %s calling function with args: %v (%d)", name, args, len(args))

	// Call the function with the prepared arguments
	out := f.Call(args)

	// Verify the number of return values matches the tool output
	if len(out) != len(tool.Output) {
		return nil, fmt.Errorf("tool %s has wrong number of return values: expected %d, got %d", name, len(tool.Output), len(out))
	}

	// Build tool MCP output from the function return values
	output := make([]McpToolOutput, 0, len(out))

	for i, o := range tool.Output {
		if o.Kind != out[i].Kind() {
			return nil, fmt.Errorf("tool %s return value %d has wrong type: expected %s, got %s", name, i, o.Kind, out[i].Kind())
		}

		switch o.Type {
		case McpToolParameterTypeString,
			McpToolParameterTypeNumber,
			McpToolParameterTypeBoolean:

			mcpOut := McpToolOutput{
				Type: McpToolOutputTypeText,
				Text: fmt.Sprint(out[i]),
			}
			output = append(output, mcpOut)
		case McpToolParameterTypeError:
			// Check if function return an error
			if !out[i].IsNil() {
				err := out[i].Interface()
				if err != nil {
					// There is an error in the return value
					// Reset the output to one single error message
					output = []McpToolOutput{
						{
							Type: McpToolOutputTypeError,
							Text: fmt.Sprint(err),
						},
					}
					s.Logf("Tool %s returns error: %s at output position %d", name, err, i)
					break
				}
			}
		default:
			return nil, fmt.Errorf("tool %s return value %d has unsupported type: %s", name, i, o.Type)
		}
	}

	s.Logf("Tool %s returned: %v", name, output)
	return output, nil
}
