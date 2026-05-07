package main

import (
	"context"
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"

	defangscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(defangscaleway.Version) //nolint:forbidigo // intentional stdout for --version flag
		return
	}
	_ = p.RunProvider(context.Background(), defangscaleway.Name, defangscaleway.Version, defangscaleway.Provider())
}
