package plugin

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Invocation
	}{
		{
			name: "metadata probe",
			args: []string{MetadataSubcommand},
			want: Invocation{Metadata: true},
		},
		{
			name: "invoked by docker",
			args: []string{"ephemeral", "run", "alpine"},
			want: Invocation{GlobalFlags: []string{}, Args: []string{"run", "alpine"}},
		},
		{
			name: "invoked by docker with global flags",
			args: []string{"--context", "remote", "-D", "ephemeral", "ls"},
			want: Invocation{GlobalFlags: []string{"--context", "remote", "-D"}, Args: []string{"ls"}},
		},
		{
			name: "standalone",
			args: []string{"run", "alpine", "echo", "ephemeral"},
			want: Invocation{Args: []string{"run", "alpine", "echo", "ephemeral"}},
		},
		{
			name: "no arguments",
			args: nil,
			want: Invocation{Args: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Parse(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestWriteMetadata(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMetadata(&buf, NewMetadata("1.2.3")); err != nil {
		t.Fatal(err)
	}

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["SchemaVersion"] != schemaVersion || got["Version"] != "1.2.3" {
		t.Errorf("unexpected metadata: %v", got)
	}
}
