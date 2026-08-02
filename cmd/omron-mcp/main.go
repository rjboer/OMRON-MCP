package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rjboer/omron-mcp/internal/logging"
	"github.com/rjboer/omron-mcp/internal/mcp"
	"github.com/rjboer/omron-mcp/internal/ui"
)

var buildCommit = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--mcp" {
		runMCPServer()
		return
	}
	runGUI(buildCommit)
}

func runMCPServer() {
	if err := mcp.NewServer().RunStdio(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runGUI(version string) {
	flags := flag.NewFlagSet("omron-mcp", flag.ExitOnError)
	projectPath := flags.String("project", "C:\\OMRON\\Data\\Solution", "Sysmac solution directory or project container")
	mcpExecutable := flags.String("mcp-executable", os.Args[0], "MCP stdio server executable; defaults to this executable")
	_ = flags.Parse(os.Args[1:])

	logger, closeLogger, err := logging.NewSessionLogger()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer closeLogger()
	if err := ui.Run(logger, *projectPath, *mcpExecutable, version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
