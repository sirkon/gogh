package gogh

const (
	goghPkg      = "/gogh"
	goFormatPkg  = "/go-format"
	runtimeStuff = "/src/runtime/"

	// ReturnZeroValues is used as a renderer's scope key to represent
	// zero values in a function returning on, most likely, errors processing.
	//
	// It is computed automatically in some cases
	ReturnZeroValues = "ReturnZeroValues"
)
