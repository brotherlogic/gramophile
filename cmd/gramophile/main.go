package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/brotherlogic/gramophile/proto"
)

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
	p := tea.NewProgram(InitialModel(client, client, client))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
