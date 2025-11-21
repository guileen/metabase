package embedding

// Stub functions for Cybertron models - these will be replaced by conditional implementations
// These exist only to prevent compilation errors when Cybertron dependencies are not available

// NewCybertronMiniLML6V2 creates a Cybertron-based all-MiniLM-L6-v2 generator
func NewCybertronMiniLML6V2(config VectorGeneratorConfig) (VectorGenerator, error) {
	// Use conditional implementation instead
	return NewConditionalCybertronMiniLML6V2(config)
}

// NewCybertronGTEsmallZh creates a Cybertron-based GTE-small-zh generator
func NewCybertronGTEsmallZh(config VectorGeneratorConfig) (VectorGenerator, error) {
	// Use conditional implementation instead
	return NewConditionalCybertronGTEsmallZh(config)
}

// NewCybertronSTSBbertTiny creates a Cybertron-based stsb-bert-tiny generator
func NewCybertronSTSBbertTiny(config VectorGeneratorConfig) (VectorGenerator, error) {
	// Use conditional implementation instead
	return NewConditionalCybertronSTSBbertTiny(config)
}
