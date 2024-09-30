package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/prototext"
)

type CLIModule struct {
	Command string
	Help    string
	Execute func(context.Context, []string) error
}

func buildContext() (context.Context, context.CancelFunc, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}

	text, err := ioutil.ReadFile(fmt.Sprintf("%v/.gramophile", dirname))
	if err != nil {
		return nil, nil, err
	}

	user := &pb.GramophileAuth{}
	err = prototext.Unmarshal(text, user)
	if err != nil {
		return nil, nil, err
	}

	mContext := metadata.AppendToOutgoingContext(context.Background(), "auth-token", user.GetToken())
	ctx, cancel := context.WithTimeout(mContext, time.Minute*30)
	return ctx, cancel, nil
}

func main() {
	t := time.Now()
	defer func() {
		if time.Since(t) > time.Second*20 {
			fmt.Printf("\nComplete in %v\n", time.Since(t))
		}
	}()
	modules := []*CLIModule{GetLogin(),
		GetGetUser(), GetGetSate(),
		GetGetRecord(),
		GetGetConfig(),
		GetWant(),
		GetClean(),
		GetGetIssue(),
		GetArrived(),
		GetListen(),
		GetKeep(),
		GetSleeve(),
		GetWidth(),
		GetOrganisation(),
		GetGoalFolder(),
		GetWantlist(),
		GetFolder(),
		GetScore(),
		GetWeight()}

	ctx, cancel, err := buildContext()
	if err != nil {
		fmt.Printf("Failure to read gramophile settings (they may not exist), falling back to no auth (%v)\n", err)
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	}
	defer cancel()

	var commands []string
	for _, module := range modules {
		commands = append(commands, module.Command)
		if module.Command == os.Args[1] {
			if len(os.Args) > 2 && os.Args[2] == "help" {
				fmt.Printf("%v\n", module.Help)
				return
			}

			if len(os.Args) > 2 {
				err := module.Execute(ctx, os.Args[2:])
				if err != nil {
					fmt.Printf("Error running %v -> %v", os.Args[1], err)
				}
				return
			} else {
				err := module.Execute(ctx, []string{})
				if err != nil {
					fmt.Printf("Error running %v -> %v", os.Args[1], err)
				}
				return
			}
		}
	}

	fmt.Printf("Unable to run command %v from %v\n", os.Args[1], commands)
}
