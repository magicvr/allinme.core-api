package version

// These can be overridden at link time via -ldflags.
var (
	Version = "0.1.0"
	Commit  = "unknown"
	BuiltAt = "unknown"
)
