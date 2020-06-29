package version

var (
	// Version is the version of the operator, replaced when releasing with the correct tag
	// DO NOT change latest to something else, the Makefile replace the pattern "latest" ;)
	Version = "latest"
)
