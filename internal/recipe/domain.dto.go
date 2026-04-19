package recipe

// IndexRecipeRequest holds the input for indexing recipe content into the vector store.
type IndexRecipeRequest struct {
	Content string `json:"content" validate:"required,min=1" example:"Classic guacamole: mash 3 ripe avocados..."`
}

// QueryRecipeRequest holds the input for querying recipes via RAG.
type QueryRecipeRequest struct {
	Question            string `json:"question"                        validate:"required,min=1,max=500" example:"How do I make guacamole?"`
	DietaryRestrictions string `json:"dietaryRestrictions,omitempty"    validate:"omitempty,max=200"      example:"vegan"`
}
