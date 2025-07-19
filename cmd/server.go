package main

import (
	"fmt"
	"os"

	"github.com/mheers/godtemplate/server"
	"github.com/spf13/cobra"
)

// server configuration flags
var (
	serverPort string
	serverCmd  = &cobra.Command{
		Use:   "server",
		Short: "starts godtemplate as server",
		Long:  `Starts a web service that handles POST requests and renders invoices as PDF`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return startServer()
		},
	}
)

func init() {
	serverCmd.Flags().StringVarP(&serverPort, "port", "p", "8080", "Port to run the server on")
	serverCmd.Flags().StringVarP(&templateFile, "template", "t", "templates/template.odt", "Path to the ODT template file")
}

func startServer() error {
	// Validate template file exists
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		return fmt.Errorf("template file not found: %s", templateFile)
	}

	// Create and start server
	srv := server.NewServer(serverPort, templateFile)
	fmt.Printf("Starting godtemplate server on port %s with template %s\n", serverPort, templateFile)

	return srv.Start()
}
