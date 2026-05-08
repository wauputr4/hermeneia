package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wauputr4/hermeneia/internal/storage"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "hermeneia:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printUsage()
		return nil
	}
	switch args[0] {
	case "init":
		path := storage.DatabasePathFromEnv()
		db, err := storage.Open(path)
		if err != nil {
			return err
		}
		defer db.Close()
		if err := storage.Migrate(ctx, db); err != nil {
			return err
		}
		fmt.Printf("initialized Hermeneia database at %s\n", path)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage() {
	fmt.Println(`Hermeneia content workflow CLI

Usage:
  hermeneia init       initialize the SQLite database

Configuration:
  HERMENEIA_DATABASE_PATH  SQLite path (default: data/hermeneia.db)`)
}
