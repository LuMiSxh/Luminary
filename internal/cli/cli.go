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

package cli

import (
	"Luminary/pkg/engine"
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
	"net/mail"
	"os"
)

// NewApp creates a new CLI application
func NewApp(engine *engine.Engine, version string) *cli.Command {
	app := &cli.Command{
		Name:                  "luminary",
		Usage:                 "A streamlined CLI tool for searching and downloading manga",
		EnableShellCompletion: true,
		Version:               version,
		Authors: []any{
			mail.Address{Name: "LuMiSxh", Address: "b71f1f@icloud.com"},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug output",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// Set error formatting mode
			if cmd.Bool("debug") {
				engine.SetDebugMode(true)
			}

			return ctx, nil
		},
		Commands: []*cli.Command{
			{
				Name:      "search",
				Aliases:   []string{"s"},
				Usage:     "Search for manga",
				ArgsUsage: "<query>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "provider",
						Aliases: []string{"p"},
						Usage:   "Provider ID (leave empty to search all)",
					},
					&cli.IntFlag{
						Name:    "limit",
						Aliases: []string{"l"},
						Usage:   "Maximum results per provider",
						Value:   10,
					},
					&cli.StringFlag{
						Name:  "fields",
						Usage: "Search in specific fields (comma-separated)",
					},
					&cli.StringFlag{
						Name:  "filter",
						Usage: "Filter results (format: field=value)",
					},
					&cli.StringFlag{
						Name:  "sort",
						Usage: "Sort results by field",
					},
				},
				Action: NewSearchCommand(engine),
			},
			{
				Name:      "info",
				Aliases:   []string{"i"},
				Usage:     "Get manga information",
				ArgsUsage: "<provider:manga-id>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "lang",
						Usage: "Filter chapters by language (comma-separated)",
					},
				},
				Action: NewInfoCommand(engine),
			},
			{
				Name:      "download",
				Aliases:   []string{"d"},
				Usage:     "Download manga chapters",
				ArgsUsage: "<provider:chapter-id> [provider:chapter-id...]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output directory",
						Value:   ".",
					},
					&cli.StringFlag{
						Name:  "format",
						Usage: "Image format (jpeg, png, webp)",
					},
					&cli.IntFlag{
						Name:  "concurrent",
						Usage: "Number of concurrent downloads",
						Value: 5,
					},
				},
				Action: NewDownloadCommand(engine),
			},
			{
				Name:    "providers",
				Aliases: []string{"p"},
				Usage:   "List available providers",
				Action:  NewProvidersCommand(engine),
			},
		},
		ExitErrHandler: func(ctx context.Context, cmd *cli.Command, err error) {
			if err != nil {
				_, err = fmt.Fprintln(os.Stderr, engine.FormatError(err))
			}
		},
	}

	return app
}
