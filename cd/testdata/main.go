package main

import (
	"os"

	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/examples/cd/program"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	compose, _ := os.ReadFile("./compose.crewai.yaml")
	pulumi.Run(program.NewRun(&defangv1.ProjectUpdate{
		Compose: compose,
	}))
}
