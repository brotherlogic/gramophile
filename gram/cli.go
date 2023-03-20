package main

import (
	"context"
	"fmt"
	"os"
)

type CLIModule struct {
	command string
	help    string
	execute func(context.Context, []string) error
}

func main() {
	modules := []*CLIModule{}

	for _, module := range modules {
		if module.command == os.Args[1] {
			if os.Args[2] == "help" {
				fmt.Printf("%v\n", module.help)
				return
			}

			err := module.execute(context.Background(), os.Args[2:])
			if err != nil {
				fmt.Printf("Error running %v -> %v", os.Args[1], err)
			}
		}
	}
}
