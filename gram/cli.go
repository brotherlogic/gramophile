package main

import (
	"context"
	"fmt"
	"os"
)

type CLIModule struct {
	Command string
	Help    string
	Execute func(context.Context, []string) error
}

func main() {
	modules := []*CLIModule{GetLogin()}

	for _, module := range modules {
		if module.Command == os.Args[1] {
			if len(os.Args) > 2 && os.Args[2] == "help" {
				fmt.Printf("%v\n", module.Help)
				return
			}

			if len(os.Args) > 2 {
				err := module.Execute(context.Background(), os.Args[2:])
				if err != nil {
					fmt.Printf("Error running %v -> %v", os.Args[1], err)
				}
			} else {
				err := module.Execute(context.Background(), []string{})
				if err != nil {
					fmt.Printf("Error running %v -> %v", os.Args[1], err)
				}
			}
		}
	}
}
