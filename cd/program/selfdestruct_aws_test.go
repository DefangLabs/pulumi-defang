package program

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCodebuildProjectFromBuildArn(t *testing.T) {
	name, arn, err := codebuildProjectFromBuildArn("arn:aws:codebuild:us-west-2:123456789012:build/defang-cd-abc:1a2b3c")
	if err != nil {
		t.Fatal(err)
	}
	if name != "defang-cd-abc" {
		t.Errorf("name = %q", name)
	}
	if arn != "arn:aws:codebuild:us-west-2:123456789012:project/defang-cd-abc" {
		t.Errorf("arn = %q", arn)
	}

	for _, bad := range []string{"", "arn:aws:codebuild:us-west-2:1:project/x", "arn:aws:codebuild:us-west-2:1:build/noid"} {
		if _, _, err := codebuildProjectFromBuildArn(bad); err == nil {
			t.Errorf("want error for %q", bad)
		}
	}
}

func TestAwsSelfDestructInput(t *testing.T) {
	environ := []string{
		"PROJECT=myproj",
		"STACK=preview",
		"DEFANG_STATE_URL=s3://bucket?region=us-west-2",
		"AWS_SECRET_ACCESS_KEY=shh", // dropped
		"PATH=/usr/bin",             // dropped
	}
	got, err := awsSelfDestructInput("defang-cd-abc", "defangio/cd:public-beta", environ)
	if err != nil {
		t.Fatal(err)
	}

	var input struct {
		ProjectName       string `json:"ProjectName"`
		ImageOverride     string `json:"ImageOverride"`
		BuildspecOverride string `json:"BuildspecOverride"`
		PullCreds         string `json:"ImagePullCredentialsTypeOverride"`
		Env               []struct {
			Name  string `json:"Name"`
			Value string `json:"Value"`
			Type  string `json:"Type"`
		} `json:"EnvironmentVariablesOverride"`
	}
	if err := json.Unmarshal([]byte(got), &input); err != nil {
		t.Fatal(err)
	}
	if input.ProjectName != "defang-cd-abc" || input.ImageOverride != "defangio/cd:public-beta" {
		t.Errorf("project/image = %q/%q", input.ProjectName, input.ImageOverride)
	}
	if input.PullCreds != "SERVICE_ROLE" {
		t.Errorf("ImagePullCredentialsTypeOverride = %q", input.PullCreds)
	}
	if !strings.Contains(input.BuildspecOverride, "/app/cd down") {
		t.Errorf("buildspec = %q", input.BuildspecOverride)
	}
	var names []string
	for _, e := range input.Env {
		names = append(names, e.Name)
		if e.Type != "PLAINTEXT" {
			t.Errorf("env %s type = %q", e.Name, e.Type)
		}
	}
	if strings.Join(names, ",") != "DEFANG_STATE_URL,PROJECT,STACK" {
		t.Errorf("env names = %v", names)
	}

	// "aws/"-prefixed curated images keep the default pull credentials.
	got, err = awsSelfDestructInput("p", "aws/codebuild/standard:7.0", environ)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "ImagePullCredentialsTypeOverride") {
		t.Errorf("curated image must not override pull credentials: %s", got)
	}
}
