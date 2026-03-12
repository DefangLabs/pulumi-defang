package main

import (
	"context"

	p "github.com/pulumi/pulumi-go-provider"

	defanggcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp"
)

func main() {
	_ = p.RunProvider(context.Background(), defanggcp.Name, defanggcp.Version, defanggcp.Provider())
}
