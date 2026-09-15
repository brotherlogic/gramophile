package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/brotherlogic/gramophile/proto"
)

var (
	Version       = "dev"
	readBuildInfo = debug.ReadBuildInfo
	semverRegex   = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
)

func isValidSemver(v string) bool {
	if v == "" || v == "(devel)" {
		return false
	}
	return semverRegex.MatchString(v)
}

func resolveVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}

	if readBuildInfo != nil {
		if info, ok := readBuildInfo(); ok && info != nil {
			if isValidSemver(info.Main.Version) {
				return info.Main.Version
			}
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" && setting.Value != "" {
					commit := setting.Value
					if len(commit) > 7 {
						commit = commit[:7]
					}
					return fmt.Sprintf("v0.0.0-dev+%s", commit)
				}
			}
		}
	}

	return "dev"
}

func getHost() string {
	host := os.Getenv("GRAMOPHILE_HOST")
	if host == "" {
		host = "gramophile-grpc.brotherlogic-backend.com:80"
	}
	return host
}

func main() {
	host := getHost()
	conn, err := grpc.NewClient(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to dial gramophile: %v", err)
	}
	defer conn.Close()

	client := pb.NewGramophileEServiceClient(conn)
	m := InitialModel(client, client, client)
	m.version = resolveVersion()
	m.updater = NewGitHubUpdater()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
