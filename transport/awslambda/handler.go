package awslambda

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/puttsk/go-mcp"
)

type TransportHandler struct{}

func (h *TransportHandler) GetSessionID(ctx context.Context, request any) (string, error) {
	if awsRequest, ok := request.(events.APIGatewayProxyRequest); ok {
		if sid, ok := awsRequest.Headers["Mcp-Session-Id"]; ok {
			return sid, nil
		} else if sid, ok := awsRequest.Headers["mcp-session-id"]; ok {
			return sid, nil
		} else {
			return "", mcp.ErrNoSessionHeader
		}
	} else {
		return "", fmt.Errorf("invalid request type: %T", request)
	}
}

func (h *TransportHandler) ProcessRequest(ctx context.Context, request any) (*mcp.McpRequest, error) {
	if awsRequest, ok := request.(events.APIGatewayProxyRequest); ok {
		req := new(mcp.McpRequest)

		d := json.NewDecoder(strings.NewReader(awsRequest.Body))
		d.UseNumber()

		err := d.Decode(req)
		if err != nil {
			return nil, fmt.Errorf("cannot decode request: %v", err)
		}
		return req, nil

	} else {
		return nil, fmt.Errorf("invalid request type: %T", request)
	}
}

func (h *TransportHandler) ProcessResponse(ctx context.Context, response *mcp.McpResponse) (any, error) {
	sess, err := mcp.GetSessionFromContext(ctx)
	if err != nil {
		return nil, err
	}

	awsResponse := events.APIGatewayProxyResponse{
		Body:       "",
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Set session ID in the response headers
	if sess.SessionID != "" {
		awsResponse.Headers["Mcp-Session-Id"] = sess.SessionID
	}

	body, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("cannot encode response: %v", err)
	}
	awsResponse.Body = string(body)

	return awsResponse, nil
}
