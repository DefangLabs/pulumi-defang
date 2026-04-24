package program

import (
	"testing"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"google.golang.org/protobuf/proto"
)

func mustMarshal(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal proto: %v", err)
	}
	return b
}

func TestExtractComposeYaml(t *testing.T) {
	compose := []byte("services:\n  web:\n    image: nginx\n")

	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:  "compose only",
			input: mustMarshal(t, &defangv1.ProjectUpdate{Compose: compose}),
			want:  string(compose),
		},
		{
			name: "compose among other fields",
			input: mustMarshal(t, &defangv1.ProjectUpdate{
				Project:   "my-project",
				Compose:   compose,
				CdVersion: "1.0.0",
				Mode:      defangv1.DeploymentMode_DEVELOPMENT,
			}),
			want: string(compose),
		},
		{
			name:    "missing compose",
			input:   mustMarshal(t, &defangv1.ProjectUpdate{Project: "my-project"}),
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "invalid protobuf",
			input:   []byte{0xff, 0xff, 0xff},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractComposeYaml(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
