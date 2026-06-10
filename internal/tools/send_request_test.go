package tools_test

import (
	"testing"

	"github.com/c0tton-fluff/caido-mcp-server/internal/replay"
	"github.com/c0tton-fluff/caido-mcp-server/internal/testutil"
	"github.com/c0tton-fluff/caido-mcp-server/internal/tools"
	caido "github.com/caido-community/sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// setupSendMocks wires the Caido 0.57 default-session send flow:
//  1. GetOrCreateSession creates an empty session (cached).
//  2. Send -> sendOnSession: GetReplaySession reports no active entry,
//     so Send falls back to creating a session seeded with the request.
//  3. StartReplayTask runs the seeded draft.
//  4. PollForEntry: GetReplaySession reports the new active entry, then
//     GetReplayEntry returns the response.
func setupSendMocks(m *testutil.MockHandler, sessionID, entryID, requestID string, statusCode int) {
	// Empty session created by GetOrCreateSession.
	m.On("CreateReplaySession", testutil.CreateReplaySessionResponse(sessionID))
	// sendOnSession sees no active entry -> fallback.
	m.On("GetReplaySession", testutil.GetReplaySessionResponse(sessionID, ""))
	// Fallback seeded session.
	m.On("CreateReplaySession", testutil.CreateReplaySessionSeededResponse(sessionID, entryID))
	m.On("StartReplayTask", testutil.StartReplayTaskResponse())
	// Poll: new active entry present, then fetch it.
	m.On("GetReplaySession", testutil.GetReplaySessionResponse(sessionID, entryID))
	m.On("GetReplayEntry", testutil.GetReplayEntryResponse(entryID, requestID, statusCode, "response body"))
}

func TestSendRequest(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		setup      func(*testutil.MockHandler)
		wantStatus int
		wantError  bool
	}{
		{
			name: "sends request and returns response",
			args: map[string]any{
				"raw":  "GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n",
				"host": "example.com",
			},
			setup: func(m *testutil.MockHandler) {
				setupSendMocks(m, "sess-1", "entry-1", "req-1", 200)
			},
			wantStatus: 200,
		},
		{
			name: "uses provided sessionId",
			args: map[string]any{
				"raw":       "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
				"host":      "example.com",
				"sessionId": "my-session",
			},
			setup: func(m *testutil.MockHandler) {
				// Provided session already has an active entry, so Send
				// updates its draft and starts the task (no new session).
				m.On("GetReplaySession", testutil.GetReplaySessionResponse("my-session", "prev-entry"))
				m.On("UpdateReplayEntryDraft", testutil.UpdateReplayEntryDraftResponse("prev-entry"))
				m.On("StartReplayTask", testutil.StartReplayTaskResponse())
				m.On("GetReplaySession", testutil.GetReplaySessionResponse("my-session", "entry-2"))
				m.On("GetReplayEntry", testutil.GetReplayEntryResponse("entry-2", "req-2", 301, ""))
			},
			wantStatus: 301,
		},
		{
			name:      "rejects empty raw",
			args:      map[string]any{"raw": ""},
			setup:     func(m *testutil.MockHandler) {},
			wantError: true,
		},
		{
			name:      "rejects raw over 1MB",
			args:      map[string]any{"raw": string(make([]byte, 1048577)), "host": "x.com"},
			setup:     func(m *testutil.MockHandler) {},
			wantError: true,
		},
		{
			name:      "rejects missing host",
			args:      map[string]any{"raw": "GET / HTTP/1.1\r\n\r\n"},
			setup:     func(m *testutil.MockHandler) {},
			wantError: true,
		},
		{
			name: "extracts host from Host header",
			args: map[string]any{
				"raw": "GET / HTTP/1.1\r\nHost: auto.example.com\r\n\r\n",
			},
			setup: func(m *testutil.MockHandler) {
				setupSendMocks(m, "sess-3", "entry-3", "req-3", 200)
			},
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replay.ResetDefaultSession("")
			t.Cleanup(func() { replay.ResetDefaultSession("") })

			env := testutil.NewMCPTestEnv(t, func(s *mcp.Server, c *caido.Client) {
				tools.RegisterSendRequestTool(s, c)
			})
			tt.setup(env.Mock)

			result := env.CallTool(t, "caido_send_request", tt.args)

			if tt.wantError {
				if !result.IsError {
					t.Fatal("expected error result")
				}
				return
			}

			output := testutil.UnmarshalToolResult[tools.SendRequestOutput](t, result)
			if output.StatusCode != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, output.StatusCode)
			}
		})
	}
}
