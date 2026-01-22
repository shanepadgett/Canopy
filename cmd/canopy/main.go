package main

import (
	"fmt"
	"os"

	"github.com/shanepadgett/canopy/internal/build"
	"github.com/shanepadgett/canopy/pkg/cli"
)

var version = "dev"

func main() {
	app := cli.New("canopy", "A fast, dependency-free static site generator", version)

	app.Add(buildCommand())
	app.Add(serveCommand())
	app.Add(newCommand())

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func buildCommand() *cli.Command {
	cmd := cli.NewCommand("build", "build [options]", "Build the site to the output directory")

	drafts := cmd.Flags.Bool("drafts", "d", false, "Include draft content")
	output := cmd.Flags.String("output", "o", "", "Output directory (overrides site.json)")

	cmd.Action = func(ctx *cli.Context) error {
		opts := build.Options{
			BuildDrafts: *drafts,
			OutputDir:   *output,
		}

		stats, err := build.Build(opts)
		if err != nil {
			return err
		}

		fmt.Printf("Built site:\n")
		fmt.Printf("  Pages:    %d\n", stats.Pages)
		fmt.Printf("  Sections: %d\n", stats.Sections)
		fmt.Printf("  Tags:     %d\n", stats.Tags)
		fmt.Printf("  Output:   %s\n", stats.Output)
		fmt.Printf("  Time:     %s\n", stats.Duration.Round(1e6))

		return nil
	}

	return cmd
}

func serveCommand() *cli.Command {
	cmd := cli.NewCommand("serve", "serve [options]", "Start a local development server")

	port := cmd.Flags.Int("port", "p", 8080, "Port to listen on")
	drafts := cmd.Flags.Bool("drafts", "d", true, "Include draft content")

	cmd.Action = func(ctx *cli.Context) error {
		fmt.Printf("Starting server on :%d (drafts=%v)...\n", *port, *drafts)
		// TODO: implement serve
		return nil
	}

	return cmd
}

func newCommand() *cli.Command {
	cmd := cli.NewCommand("new", "new <type> <title>", "Create new content")

	// Subcommand: new post
	postCmd := cli.NewCommand("post", "new post <title>", "Create a new blog post")
	postCmd.Action = func(ctx *cli.Context) error {
		if len(ctx.Args) < 1 {
			return fmt.Errorf("title required: canopy new post <title>")
		}
		title := ctx.Args[0]
		fmt.Printf("Creating new post: %q\n", title)
		// TODO: implement new post
		return nil
	}

	// Subcommand: new guide
	guideCmd := cli.NewCommand("guide", "new guide <title>", "Create a new guide")
	guideCmd.Action = func(ctx *cli.Context) error {
		if len(ctx.Args) < 1 {
			return fmt.Errorf("title required: canopy new guide <title>")
		}
		title := ctx.Args[0]
		fmt.Printf("Creating new guide: %q\n", title)
		// TODO: implement new guide
		return nil
	}

	// Subcommand: new page
	pageCmd := cli.NewCommand("page", "new page <title>", "Create a new standalone page")
	pageCmd.Action = func(ctx *cli.Context) error {
		if len(ctx.Args) < 1 {
			return fmt.Errorf("title required: canopy new page <title>")
		}
		title := ctx.Args[0]
		fmt.Printf("Creating new page: %q\n", title)
		// TODO: implement new page
		return nil
	}

	cmd.AddSubcommand(postCmd)
	cmd.AddSubcommand(guideCmd)
	cmd.AddSubcommand(pageCmd)

	return cmd
}
