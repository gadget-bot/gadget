// Package conf holds build-time configuration injected via ldflags.
package conf

// Executable is the name of the binary, set at build time via -ldflags.
var Executable = "gadget"

// GitVersion is the git version string, set at build time via -ldflags.
var GitVersion = "dev"
