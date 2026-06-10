package tools

import (
	"context"
	"fmt"

	caido "github.com/caido-community/sdk-go"
	gen "github.com/caido-community/sdk-go/graphql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CreateReplaySessionInput struct {
	Name            string `json:"name,omitempty" jsonschema:"Session name (applied via rename after creation)"`
	CollectionID    string `json:"collectionId,omitempty" jsonschema:"Collection ID to assign the session to"`
	RequestSourceID string `json:"requestSourceId,omitempty" jsonschema:"Existing request ID to seed the session with"`
}

type CreateReplaySessionOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func createReplaySessionHandler(
	client *caido.Client,
) func(context.Context, *mcp.CallToolRequest, CreateReplaySessionInput) (*mcp.CallToolResult, CreateReplaySessionOutput, error) {
	return func(
		ctx context.Context,
		req *mcp.CallToolRequest,
		input CreateReplaySessionInput,
	) (*mcp.CallToolResult, CreateReplaySessionOutput, error) {
		createInput := &gen.CreateReplaySessionInput{}

		if input.CollectionID != "" {
			createInput.CollectionId = &input.CollectionID
		}
		if input.RequestSourceID != "" {
			createInput.RequestSource = &gen.RequestSourceInput{
				Id: &input.RequestSourceID,
			}
		}

		sessionID, _, err := client.Replay.CreateSession(ctx, createInput)
		if err != nil {
			return nil, CreateReplaySessionOutput{}, fmt.Errorf("create session: %w", err)
		}

		output := CreateReplaySessionOutput{ID: sessionID}

		if input.Name != "" {
			if _, err := client.Replay.RenameSession(
				ctx, sessionID, input.Name,
			); err != nil {
				return nil, CreateReplaySessionOutput{}, fmt.Errorf("rename session: %w", err)
			}
			output.Name = input.Name
		}

		return nil, output, nil
	}
}

func RegisterCreateReplaySessionTool(
	server *mcp.Server, client *caido.Client,
) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "caido_create_replay_session",
		Description: `Create a named replay session. Optionally seed it with an existing request and assign to a collection. Use this to organize replay work by target or task.`,
	}, createReplaySessionHandler(client))
}
