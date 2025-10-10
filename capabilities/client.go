package capabilities

// ClientCapabilities represents the capabilities declared by the client.
type ClientCapabilities struct {
	Elicitation *ElicitationCapabilities
	Sampling    *SamplingCapabilities
	Roots       *RootsCapabilities
}

// ElicitationCapabilities indicates the client supports elicitation.
type ElicitationCapabilities struct{}

// SamplingCapabilities indicates the client supports LLM sampling.
type SamplingCapabilities struct{}

// RootsCapabilities indicates the client supports roots.
type RootsCapabilities struct {
	// ListChanged indicates whether the client supports notifications for changes to the roots list.
	ListChanged bool
}
