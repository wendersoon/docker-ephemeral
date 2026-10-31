// Package plugin implements the Docker CLI plugin protocol.
//
// See https://github.com/docker/cli/blob/master/docs/extend/cli_plugins.md.
package plugin

import (
	"encoding/json"
	"io"
	"slices"
	"strings"
)

const (
	// Name is the subcommand under which the plugin is exposed: docker <Name>.
	Name = "ephemeral"
	// MetadataSubcommand is invoked by the Docker CLI to discover the plugin.
	MetadataSubcommand = "docker-cli-plugin-metadata"

	schemaVersion = "0.1.0"
	vendor        = "wendersoon"
	url           = "https://github.com/wendersoon/docker-ephemeral"
)

// Metadata is the payload answered to MetadataSubcommand.
type Metadata struct {
	SchemaVersion    string `json:"SchemaVersion"`
	Vendor           string `json:"Vendor"`
	Version          string `json:"Version"`
	ShortDescription string `json:"ShortDescription"`
	URL              string `json:"URL"`
}

// NewMetadata returns the plugin metadata for the given version.
func NewMetadata(version string) Metadata {
	return Metadata{
		SchemaVersion:    schemaVersion,
		Vendor:           vendor,
		Version:          version,
		ShortDescription: "Run containers that leave no trace behind",
		URL:              url,
	}
}

// WriteMetadata encodes m as JSON into w.
func WriteMetadata(w io.Writer, m Metadata) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

// Invocation describes how the binary was called.
type Invocation struct {
	// Metadata is true when the Docker CLI is probing the plugin.
	Metadata bool
	// GlobalFlags are Docker global options given before the plugin name
	// (e.g. --context), which must be forwarded to every docker call.
	GlobalFlags []string
	// Args are the arguments addressed to the plugin itself.
	Args []string
}

// globalFlagsWithValue lists Docker global options that consume the next argument.
var globalFlagsWithValue = []string{
	"-c", "--context",
	"-H", "--host",
	"-l", "--log-level",
	"--config",
	"--tlscacert", "--tlscert", "--tlskey",
}

// Parse interprets the raw process arguments (without the program name).
//
// When run through the Docker CLI the arguments look like
// "[global flags] ephemeral [args]"; when run standalone they are the plugin
// arguments directly.
func Parse(args []string) Invocation {
	if len(args) > 0 && args[0] == MetadataSubcommand {
		return Invocation{Metadata: true}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == Name {
			return Invocation{
				GlobalFlags: slices.Clone(args[:i]),
				Args:        slices.Clone(args[i+1:]),
			}
		}
		if !strings.HasPrefix(arg, "-") {
			break
		}
		if slices.Contains(globalFlagsWithValue, arg) {
			i++
		}
	}

	return Invocation{Args: slices.Clone(args)}
}
