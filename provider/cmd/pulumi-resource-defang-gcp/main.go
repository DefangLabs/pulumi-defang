package main

import (
	"context"
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"

	defanggcp "github.com/DefangLabs/pulumi-defang/provider/defanggcp"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(defanggcp.Version) //nolint:forbidigo // intentional stdout for --version flag
		return
	}
	_ = p.RunProvider(context.Background(), defanggcp.Name, defanggcp.Version, defanggcp.Provider())
}
