package alexa

// Speech responses are all plain text; Alexa speaks them verbatim.
func Tell(text string) Response {
	return Response{
		Version: "1.0",
		Response: Resp{
			OutputSpeech:     &OutputSpeech{Type: "PlainText", Text: text},
			ShouldEndSession: true,
		},
	}
}

// Ask keeps the session open for a follow-up.
func Ask(text, reprompt string) Response {
	return Response{
		Version: "1.0",
		Response: Resp{
			OutputSpeech:     &OutputSpeech{Type: "PlainText", Text: text},
			Reprompt:         &Reprompt{OutputSpeech: OutputSpeech{Type: "PlainText", Text: reprompt}},
			ShouldEndSession: false,
		},
	}
}

// EndSession returns an empty response that closes the session (used for
// SessionEndedRequest and AMAZON.StopIntent without extra speech).
func EndSession() Response {
	return Response{Version: "1.0", Response: Resp{ShouldEndSession: true}}
}

// Response is the envelope Alexa expects back.
type Response struct {
	Version           string            `json:"version"`
	SessionAttributes map[string]string `json:"sessionAttributes,omitempty"`
	Response          Resp              `json:"response"`
}

// Resp is the inner directive object.
type Resp struct {
	OutputSpeech     *OutputSpeech `json:"outputSpeech,omitempty"`
	Card             *Card         `json:"card,omitempty"`
	Reprompt         *Reprompt     `json:"reprompt,omitempty"`
	ShouldEndSession bool          `json:"shouldEndSession"`
}

type OutputSpeech struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Reprompt struct {
	OutputSpeech OutputSpeech `json:"outputSpeech"`
}

type Card struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
