package alexa

import (
	"encoding/json"
	"testing"
)

func TestParseRequest(t *testing.T) {
	raw := `{
		"version": "1.0",
		"session": {
			"sessionId": "amzn1.echo-api.session.abc",
			"application": {"applicationId": "amzn1.ask.skill.123"},
			"user": {"userId": "amzn1.ask.account.user1"}
		},
		"context": {
			"System": {
				"application": {"applicationId": "amzn1.ask.skill.123"},
				"user": {"userId": "amzn1.ask.account.user1"}
			}
		},
		"request": {
			"type": "IntentRequest",
			"requestId": "amzn1.echo-api.request.xyz",
			"timestamp": "2026-08-12T00:00:00Z",
			"locale": "en-US",
			"intent": {
				"name": "MacCommandIntent",
				"slots": {
					"Action": {"name": "Action", "value": "open spotify"}
				}
			}
		}
	}`

	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if req.Version != "1.0" {
		t.Errorf("Version = %q", req.Version)
	}
	if req.Request.Type != RequestTypeIntent {
		t.Errorf("Type = %q, want IntentRequest", req.Request.Type)
	}
	if req.Request.Intent.Name != IntentMacCommand {
		t.Errorf("Intent.Name = %q", req.Request.Intent.Name)
	}
	if got := req.Request.Intent.Slots["Action"].Value; got != "open spotify" {
		t.Errorf("Action slot = %q", got)
	}
	if req.Context.System.Application.ApplicationID != "amzn1.ask.skill.123" {
		t.Errorf("context skill ID = %q", req.Context.System.Application.ApplicationID)
	}
}

func TestParseRequestNoIntent(t *testing.T) {
	raw := `{"version":"1.0","request":{"type":"LaunchRequest","requestId":"r1","timestamp":"2026-08-12T00:00:00Z"}}`
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if req.Request.Type != RequestTypeLaunch {
		t.Errorf("Type = %q, want LaunchRequest", req.Request.Type)
	}
}
