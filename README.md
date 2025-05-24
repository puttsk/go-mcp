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

var server *mcp.McpServer

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

  // Processing request from AWS Lambda
  resp, err := server.ProcessRequest(ctx, request)
  if err != nil {
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
  // Initialize the MCP server with in-memory session manager and AWS Lambda transport handler
  server, _ = mcp.NewMcpServer("mcptest", "1.0.0", mcp.McpProtocol2025_30_26)
  server.LogLevel = mcp.LogLevelDebug
  server.TransportHandler = &awslambda.TransportHandler{} // Setup transport handler
  server.SessionManager = memory.NewSessionManager() // Setup in-memory session manager

  // Register tool "add" with the server
  server.RegisterTool("add", "Tool for adding two numbers",
    func(a, b int) int { return a + b },
    mcp.McpToolParameter{Name: "a", Description: "First number"},
    mcp.McpToolParameter{Name: "b", Description: "Second number"},
  )
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
* Tool inputs are limited to **scalar types**: `number`, `string`, and `boolean`.
* Tool outputs are limited to **text only**.
