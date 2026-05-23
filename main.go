package main

import (
	"context"
	"log"
	"os"

	"github.com/Delta-Daemon/mcp-server/auth"
	"github.com/Delta-Daemon/mcp-server/client"
	"github.com/Delta-Daemon/mcp-server/prompts"
	"github.com/Delta-Daemon/mcp-server/resources"
	"github.com/Delta-Daemon/mcp-server/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	log.SetOutput(os.Stderr)
	auth.SetServeHandler(runServer)
	os.Exit(auth.RunCLI(os.Args[1:]))
}

func runServer() int {
	api := client.New()
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "deltadaemon",
		Version: "1.0.0",
		Title:   "DeltaDaemon Forecast Accuracy",
	}, nil)

	tools.Register(server, api)
	resources.Register(server, api)
	prompts.Register(server)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
	return 0
}
