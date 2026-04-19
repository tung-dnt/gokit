package recipe

// indexResponse is the HTTP JSON response for a document indexing operation.
type indexResponse struct {
	Success          bool `json:"success"`
	DocumentsIndexed int  `json:"documents_indexed"`
}

// queryResponse is the HTTP JSON response for a RAG query.
// Also used as the Dotprompt output schema.
type queryResponse struct {
	Answer  string   `json:"answer"`
	Sources []string `json:"sources,omitempty"`
}
