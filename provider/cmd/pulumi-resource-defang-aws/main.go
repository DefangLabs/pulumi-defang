package main

import (
	"context"
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"

	defangaws "github.com/DefangLabs/pulumi-defang/provider/defangaws"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(defangaws.Version) //nolint:forbidigo // intentional stdout for --version flag
		return
	}
	_ = p.RunProvider(context.Background(), defangaws.Name, defangaws.Version, defangaws.Provider())
}
