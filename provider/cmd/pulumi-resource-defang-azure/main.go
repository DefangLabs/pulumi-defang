package main

import (
	"context"
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"

	defangazure "github.com/DefangLabs/pulumi-defang/provider/defangazure"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(defangazure.Version) //nolint:forbidigo // intentional stdout for --version flag
		return
	}
	_ = p.RunProvider(context.Background(), defangazure.Name, defangazure.Version, defangazure.Provider())
}
