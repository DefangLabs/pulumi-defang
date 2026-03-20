package main

import (
	"context"

	p "github.com/pulumi/pulumi-go-provider"

	defangazure "github.com/DefangLabs/pulumi-defang/provider/defangazure"
)

func main() {
	_ = p.RunProvider(context.Background(), defangazure.Name, defangazure.Version, defangazure.Provider())
}
