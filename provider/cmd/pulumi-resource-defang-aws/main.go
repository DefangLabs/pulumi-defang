package main

import (
	"context"

	p "github.com/pulumi/pulumi-go-provider"

	defangaws "github.com/DefangLabs/pulumi-defang/provider/defangaws"
)

func main() {
	_ = p.RunProvider(context.Background(), defangaws.Name, defangaws.Version, defangaws.Provider())
}
