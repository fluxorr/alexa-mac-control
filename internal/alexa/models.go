// Package alexa implements the Alexa Custom Skill side of the controller
// (PRD §17-§18): request models, intent parsing, response generation and
// request verification. This layer never knows how commands execute; it only
// maps intents to allowlisted command names (PRD §6).
package alexa

import "encoding/json"

// Request is the top-level envelope Alexa posts to the skill endpoint.
// Only the fields the controller needs are modeled; unknown fields are
// tolerated by the decoder.
type Request struct {
	Version string        `json:"version"`
	Session Session       `json:"session"`
	Context Context       `json:"context"`
	Request IntentRequest `json:"request"`
}

// Session carries session-scoped identity.
type Session struct {
	SessionID   string `json:"sessionId"`
	Application struct {
		ApplicationID string `json:"applicationId"`
	} `json:"application"`
	User struct {
		UserID string `json:"userId"`
	} `json:"user"`
}

// Context carries system-level identity, used for skill ID verification.
type Context struct {
	System struct {
		Application struct {
			ApplicationID string `json:"applicationId"`
		} `json:"application"`
		User struct {
			UserID string `json:"userId"`
		} `json:"user"`
	} `json:"System"`
}

// IntentRequest is the typed request body, discriminated on Type.
type IntentRequest struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	Timestamp string `json:"timestamp"`
	Locale    string `json:"locale"`
	Intent    Intent `json:"intent"`
	Reason    string `json:"reason"`
	Error     *Error `json:"error,omitempty"`
}

// Intent names the skill handles (PRD §17).
const (
	IntentMacCommand    = "MacCommandIntent"
	IntentMacSearch     = "MacSearchIntent"
	IntentMacFileSearch = "MacFileSearchIntent"
	IntentMacStatus     = "MacStatusIntent"
	IntentHelp          = "AMAZON.HelpIntent"
	IntentCancel        = "AMAZON.CancelIntent"
	IntentStop          = "AMAZON.StopIntent"
	IntentFallback      = "AMAZON.FallbackIntent"
)

// Request types Alexa may send.
const (
	RequestTypeLaunch     = "LaunchRequest"
	RequestTypeIntent     = "IntentRequest"
	RequestTypeSessionEnd = "SessionEndedRequest"
)

// Intent is a parsed user intent with its slots.
type Intent struct {
	Name  string          `json:"name"`
	Slots map[string]Slot `json:"slots"`
}

// Slot holds a single intent slot. Resolutions is kept raw: the controller
// only needs the transcribed value.
type Slot struct {
	Name        string          `json:"name"`
	Value       string          `json:"value"`
	Resolutions json.RawMessage `json:"resolutions,omitempty"`
}

// Error describes a failed Alexa request.
type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
