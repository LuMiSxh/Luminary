// Luminary: A streamlined CLI tool for searching and downloading manga.
// Copyright (C) 2025 Luca M. Schmidt (LuMiSxh)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"Luminary/internal/providers"
	"Luminary/internal/rpc" // Internal RPC services
	"Luminary/pkg/engine"
	"bufio"
	"io"
	"log"
	gorpc "net/rpc" // Alias for Go's standard RPC package
	"net/rpc/jsonrpc"
	"os"
)

// Version is set during build using -ldflags
var Version = "0.0.0-dev"

// stdInOutReadWriteCloser adapts os.Stdin and os.Stdout to io.ReadWriteCloser.
type stdInOutReadWriteCloser struct {
	reader io.Reader // Use io.Reader for flexibility, will be a *bufio.Reader for stdin
	writer io.Writer
	closer io.Closer // Optional closer
}

func (s *stdInOutReadWriteCloser) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

func (s *stdInOutReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = s.writer.Write(p)
	if err != nil {
		return n, err
	}
	// Attempt to flush if writer supports it (e.g., *os.File)
	// This is important for interactive use or when piping.
	if f, ok := s.writer.(interface{ Sync() error }); ok {
		_ = f.Sync() // Best effort, ignore error as not much can be done.
	}
	return n, err
}

func (s *stdInOutReadWriteCloser) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	// Stdin/Stdout are not typically closed by the application.
	return nil
}

// registerProviders registers all available manga source providers with the engine.
// This function should mirror the provider registration in your CLI's main.go.
func registerProviders(e *engine.Engine) {
	// Example: Register MangaDex provider
	// Ensure NewMangadexProvider and other provider constructors are accessible
	// from the 'internal/providers' package.
	if err := e.RegisterProvider(providers.NewMangadexProvider(e)); err != nil {
		e.Logger.Error("Failed to register MangaDex provider in RPC: %v", err)
	}

	// Example: Register Madara provider (KissManga was an example name)
	if err := e.RegisterProvider(providers.NewMadaraProvider(e)); err != nil {
		e.Logger.Error("Failed to register Madara provider in RPC: %v", err)
	}
	// Add other providers here as needed
}

func main() {
	// Initialize the Luminary engine
	appEngine := engine.New()

	// Register all available providers
	registerProviders(appEngine)

	// Create the main services container, passing the engine and version
	rpcServicesContainer := rpc.NewServices(appEngine, Version)

	// Instantiate individual services, passing the container
	versionService := &rpc.VersionService{Services: rpcServicesContainer}
	providersService := &rpc.ProvidersService{Services: rpcServicesContainer}
	searchService := &rpc.SearchService{Services: rpcServicesContainer}
	infoService := &rpc.InfoService{Services: rpcServicesContainer}
	downloadService := &rpc.DownloadService{Services: rpcServicesContainer}
	listService := &rpc.ListService{Services: rpcServicesContainer}

	// Register services with the Go RPC server
	// It's conventional to use "ServiceName.MethodName" for JSON-RPC.
	// The `RegisterName` method allows specifying the service name.
	server := gorpc.NewServer()
	if err := server.RegisterName("VersionService", versionService); err != nil {
		log.Fatalf("RPC: Failed to register VersionService: %v", err)
	}
	if err := server.RegisterName("ProvidersService", providersService); err != nil {
		log.Fatalf("RPC: Failed to register ProvidersService: %v", err)
	}
	if err := server.RegisterName("SearchService", searchService); err != nil {
		log.Fatalf("RPC: Failed to register SearchService: %v", err)
	}
	if err := server.RegisterName("InfoService", infoService); err != nil {
		log.Fatalf("RPC: Failed to register InfoService: %v", err)
	}
	if err := server.RegisterName("DownloadService", downloadService); err != nil {
		log.Fatalf("RPC: Failed to register DownloadService: %v", err)
	}
	if err := server.RegisterName("ListService", listService); err != nil {
		log.Fatalf("RPC: Failed to register ListService: %v", err)
	}

	// Set up communication over stdin/stdout
	// Use a buffered reader for stdin for efficiency.
	codec := jsonrpc.NewServerCodec(&stdInOutReadWriteCloser{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	})

	appEngine.Logger.Info("Starting RPC Server on stdin/stdout")

	// Start serving RPC requests. This will block.
	server.ServeCodec(codec)

	appEngine.Logger.Info("RPC Server exited normally")
}
