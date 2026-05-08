package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wauputr4/hermeneia/internal/storage"
)

func main() {
	cmd := command{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	if err := cmd.run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "hermeneia:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cmd := command{
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	return cmd.run(ctx, args)
}

type command struct {
	stdout io.Writer
	stderr io.Writer
}

func (c command) run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		c.printUsage()
		return nil
	}

	name := args[0]
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("unknown flag %q; run \"hermeneia help\" for usage", name)
	}

	switch args[0] {
	case "init":
		if len(args) > 1 {
			return fmt.Errorf("init does not accept arguments yet; configure the database with HERMENEIA_DATABASE_PATH")
		}
		path := storage.DatabasePathFromEnv()
		db, err := storage.Open(path)
		if err != nil {
			return fmt.Errorf("open database %q: %w", path, err)
		}
		defer db.Close()
		if err := storage.Migrate(ctx, db); err != nil {
			return fmt.Errorf("migrate database %q: %w", path, err)
		}
		fmt.Fprintf(c.stdout, "initialized Hermeneia database at %s\n", path)
		return nil
	case "create", "list", "show", "revise", "render":
		return fmt.Errorf("%s is part of the MVP command surface but is not implemented yet", args[0])
	default:
		return fmt.Errorf("unknown command %q; run \"hermeneia help\" for usage", args[0])
	}
}

func (c command) printUsage() {
	fmt.Fprintln(c.stdout, `Hermeneia content workflow CLI

Usage:
  hermeneia init              initialize the SQLite database
  hermeneia create            create a content run (planned)
  hermeneia list              list content runs (planned)
  hermeneia show              show a content run (planned)
  hermeneia revise            create a new brief revision (planned)
  hermeneia render            render/export run artifacts (planned)

Help:
  hermeneia help
  hermeneia --help

Configuration:
  HERMENEIA_DATABASE_PATH  SQLite path (default: data/hermeneia.db)`)
}
