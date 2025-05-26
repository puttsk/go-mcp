# go-mcp
Go module for running MCP server on AWS Lambda.

## Install

```
go get -u github.com/puttsk/go-mcp
```

## Usage

### AWS Lambda function 

This example demonstrates how to create an MCP server on AWS Lambda with a simple tool named `add`, which adds two integers. The server uses an **in-memory session manager** along with a **Lambda transport handler**. 

> **Note**: The in-memory session manager used in this example is intended for testing purposes only. Since AWS Lambda instances are stateless and can be terminated at any time, session data cannot be shared across instances and will be reset when the function ends.


```go
// main.go
package main

import (
  "context"
  "fmt"
  "log"
  "net/http"

  "github.com/aws/aws-lambda-go/events"
  "github.com/aws/aws-lambda-go/lambda"
  "github.com/puttsk/go-mcp"
  "github.com/puttsk/go-mcp/session/memory"
  "github.com/puttsk/go-mcp/transport/awslambda"
)

var headers map[string]string = map[string]string{
  "Content-Type":                 "application/json",
  "Access-Control-Allow-Headers": "Content-Type",
  "Access-Control-Allow-Origin":  "*",    // Allow from anywhere
  "Access-Control-Allow-Methods": "POST", // Allow only GET request
}

var server *mcp.McpServer

func simpleFunc(ctx context.Context, a, b int) int {
  return a + b
}

func scalarTypeFunc(a string, b float64, c bool) (string, error) {
  return fmt.Sprintf("String: %s\nNumber: %0.2f\nBoolean: %t", a, b, c), nil
}

func contextFunc(ctx context.Context, a int, b int) (string, error) {
  sessionId, _ := mcp.GetSessionFromContext(ctx)
  request, _ := mcp.GetRequestIDFromContext(ctx)

  return fmt.Sprintf("Session ID: %s\nRequest ID: %s\nSum: %d", sessionId.SessionID, request, a+b), nil
}

func errorFunc(ctx context.Context, a int, b int) (int, error) {
  return 0, fmt.Errorf("this is an error")
}

func imageFunc(ctx context.Context, img mcp.McpImage) (mcp.McpImage, error) {
  return img, nil
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
  // Return Method Not Allowed for non-POST requests
  if request.HTTPMethod != http.MethodPost {
    log.Printf("Invalid method: %s", request.HTTPMethod)
    return events.APIGatewayProxyResponse{
      Body:       "Method not allowed",
      Headers:    headers,
      StatusCode: http.StatusMethodNotAllowed,
    }, nil
  }

  log.Printf("Request: %#v", request.Body)
  log.Printf("Headers: %#v", request.Headers)

  resp, err := server.ProcessRequest(ctx, request)
  if err != nil {
    log.Printf("Error processing request: %v", err)
    return events.APIGatewayProxyResponse{
      Body:       "Error processing request",
      Headers:    headers,
      StatusCode: http.StatusInternalServerError,
    }, nil
  }

  if response, ok := resp.(events.APIGatewayProxyResponse); ok {
    return response, nil
  } else {
    return events.APIGatewayProxyResponse{
      Body:       "Invalid response type",
      Headers:    headers,
      StatusCode: http.StatusInternalServerError,
    }, nil
  }
}

func init() {
  // Initialize the MCP server, session manager, and transport handler
  server, _ = mcp.NewMcpServer("botioapi", "1.0.0", mcp.McpProtocol2025_30_26)
  server.LogLevel = mcp.LogLevelDebug
  server.TransportHandler = &awslambda.TransportHandler{}
  server.SessionManager = memory.NewSessionManager()

  server.RegisterTool("simple_func", simpleFunc, mcp.McpToolParameter{Name: "a", Description: "First integer"}, mcp.McpToolParameter{Name: "b", Description: "Second integer"})
  server.SetToolDescription("simple_func", "A simple function that adds two integers and returns the result.")

  server.RegisterTool("scalar_type_func", scalarTypeFunc, mcp.McpToolParameter{Name: "a"}, mcp.McpToolParameter{Name: "b"}, mcp.McpToolParameter{Name: "c"})
  server.SetToolDescription("scalar_type_func", "A function that takes a string, a float, and a boolean, and returns a formatted string.")

  server.RegisterTool("context_func", contextFunc, mcp.McpToolParameter{Name: "a", Description: "First integer"}, mcp.McpToolParameter{Name: "b", Description: "Second integer"})
  server.SetToolDescription("context_func", "A function that uses context to return session and request IDs along with the sum of two integers.")

  server.RegisterTool("error_func", errorFunc, mcp.McpToolParameter{Name: "a", Description: "First integer"}, mcp.McpToolParameter{Name: "b", Description: "Second integer"})
  server.SetToolDescription("error_func", "A function that returns an error when called.")

  server.RegisterTool("image_func", imageFunc, mcp.McpToolParameter{Name: "img", Description: "Image"})
  server.SetToolDescription("image_func", "A function that returns the image passed to it without modification.")
}

func main() {
  lambda.Start(handler)
}
```

### Tool Functions

Tool functions can accept a Go context as their first parameter. You can retrieve the current session and MCP request ID using the following helper functions:

* `GetSessionFromContext(ctx)`
* `GetRequestIDFromContext(ctx)`

These functions allow you to access session-specific and request-specific information within your tool logic.

## Known Limitations

* Only support **streamable HTTP** transport; **Server-Sent Event (SSE)** and **stdin** are not support
* Only support following MCP methods
  * `initailize`
  * `notifications/initialized`
  * `tools/list`
  * `tools/call`
* Tool inputs are limited to **scalar types**: `number`, `string`, `boolean`, and `image`.
* Tool outputs are limited to **text** and **image**.
