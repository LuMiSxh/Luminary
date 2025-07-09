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
	_ "Luminary/internal/providers" // Import for side effects (auto-registration)
	"Luminary/internal/rpc"
	"Luminary/pkg/engine"
	"Luminary/pkg/provider/registry"
	"bufio"
	"context"
	"fmt"
	"io"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var (
	Version = "dev"
)

// GetRuntimeInfo returns runtime information
func GetRuntimeInfo() rpc.RuntimeInfo {
	return rpc.RuntimeInfo{
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// stdInOutReadWriteCloser wraps stdin/stdout for JSON-RPC
type stdInOutReadWriteCloser struct {
	reader io.Reader
	writer io.Writer
	closer io.Closer
}

func (s *stdInOutReadWriteCloser) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

func (s *stdInOutReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = s.writer.Write(p)
	if err == nil {
		// Ensure output is flushed
		if flusher, ok := s.writer.(interface{ Flush() error }); ok {
			err := flusher.Flush()
			if err != nil {
				return 0, err
			}
		}
	}
	return n, err
}

func (s *stdInOutReadWriteCloser) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

func main() {
	// Initialize the Luminary engine
	appEngine := engine.New()
	defer func(appEngine *engine.Engine) {
		err := appEngine.Shutdown()
		if err != nil {

		}
	}(appEngine)

	// Load all registered providers
	if err := registry.LoadAll(appEngine); err != nil {
		appEngine.Logger.Error("Failed to load providers: %v", err)
		// Continue anyway - we can still serve RPC without providers
	}

	// Initialize providers
	ctx := context.Background()
	if err := appEngine.InitializeProviders(ctx); err != nil {
		appEngine.Logger.Error("Failed to initialize providers: %v", err)
		// Continue anyway
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		appEngine.Logger.Info("RPC server shutting down...")
		err := appEngine.Shutdown()
		if err != nil {
			return
		}
		os.Exit(0)
	}()

	// Create the RPC server with services
	rpcServer := rpc.NewServer(appEngine, Version)

	// Set up JSON-RPC over stdin/stdout
	rwc := &stdInOutReadWriteCloser{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}

	// Log startup
	appEngine.Logger.Info("Luminary RPC server v%s started", Version)
	appEngine.Logger.Info("Loaded %d providers", appEngine.ProviderCount())

	// Log initial status to stderr (won't interfere with JSON-RPC)
	_, err := fmt.Fprintf(os.Stderr, "Luminary RPC v%s ready with %d providers\n", Version, appEngine.ProviderCount())
	if err != nil {
		return
	}

	// Start serving JSON-RPC (this blocks)
	codec := jsonrpc.NewServerCodec(rwc)
	rpcServer.ServeCodec(codec)

	// If we get here, the connection was closed
	appEngine.Logger.Info("RPC connection closed")
}
