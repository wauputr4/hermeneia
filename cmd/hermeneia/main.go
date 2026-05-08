package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/wauputr4/hermeneia/internal/runfiles"
	"github.com/wauputr4/hermeneia/internal/storage"
	"github.com/wauputr4/hermeneia/internal/workflow"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "hermeneia:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cmd := command{
		stdout: os.Stdout,
	}
	return cmd.run(ctx, args)
}

type command struct {
	stdout   io.Writer
	runsRoot string
	now      func() time.Time
	newID    func(prefix, seed string) string
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
	case "create":
		return c.create(ctx, args[1:])
	case "list":
		return c.list(ctx, args[1:])
	case "show":
		return c.show(ctx, args[1:])
	case "revise":
		return c.revise(ctx, args[1:])
	case "render":
		return c.render(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q; run \"hermeneia help\" for usage", args[0])
	}
}

func (c command) create(ctx context.Context, args []string) error {
	fs := c.flagSet("create")
	var input workflow.CreateInput
	fs.StringVar(&input.Topic, "topic", "", "content topic")
	fs.StringVar(&input.ContentType, "type", "carousel", "content type: carousel or short_video")
	fs.StringVar(&input.TemplateID, "template", "", "template id")
	fs.StringVar(&input.Tone, "tone", "", "brief tone")
	fs.StringVar(&input.Platform, "platform", "", "target platform")
	fs.StringVar(&input.TargetAudience, "audience", "", "target audience")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if input.Topic == "" && fs.NArg() > 0 {
		input.Topic = strings.Join(fs.Args(), " ")
	}
	return c.withService(ctx, func(s workflow.Service) error {
		result, err := s.CreateRun(ctx, input)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "created run %s\nbrief %s\n", result.Run.ID, result.BriefPath)
		return nil
	})
}

func (c command) list(ctx context.Context, args []string) error {
	fs := c.flagSet("list")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("list does not accept positional arguments")
	}
	return c.withService(ctx, func(s workflow.Service) error {
		runs, err := s.ListRuns(ctx)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			fmt.Fprintln(c.stdout, "no content runs found")
			return nil
		}
		for _, run := range runs {
			fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\n", run.ID, run.ContentType, run.TemplateID, run.Topic)
		}
		return nil
	})
}

func (c command) show(ctx context.Context, args []string) error {
	runID, err := parseRunArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s workflow.Service) error {
		details, err := s.ShowRun(ctx, runID)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "run: %s\n", details.Run.ID)
		fmt.Fprintf(c.stdout, "topic: %s\n", details.Run.Topic)
		fmt.Fprintf(c.stdout, "type: %s\n", details.Run.ContentType)
		fmt.Fprintf(c.stdout, "template: %s\n", details.Run.TemplateID)
		fmt.Fprintf(c.stdout, "brief_versions: %d\n", len(details.Briefs))
		fmt.Fprintf(c.stdout, "revisions: %d\n", len(details.Revisions))
		fmt.Fprintf(c.stdout, "artifacts: %d\n", len(details.Artifacts))
		for _, artifact := range details.Artifacts {
			fmt.Fprintf(c.stdout, "- %s %s\n", artifact.Kind, artifact.Path)
		}
		return nil
	})
}

func (c command) revise(ctx context.Context, args []string) error {
	runID, instruction, err := parseReviseArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s workflow.Service) error {
		result, err := s.ReviseRun(ctx, runID, instruction)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "revised run %s to brief v%d\nbrief %s\n", result.Run.ID, result.Brief.Version, result.BriefPath)
		return nil
	})
}

func (c command) render(ctx context.Context, args []string) error {
	runID, err := parseRunArgs(args)
	if err != nil {
		return err
	}
	return c.withService(ctx, func(s workflow.Service) error {
		result, err := s.RenderRun(ctx, runID)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "rendered run %s from brief v%d\n", result.Run.ID, result.Brief.Version)
		for _, artifact := range result.Artifacts {
			fmt.Fprintf(c.stdout, "- %s %s\n", artifact.Kind, artifact.Path)
		}
		return nil
	})
}

func (c command) withService(ctx context.Context, fn func(workflow.Service) error) error {
	path := storage.DatabasePathFromEnv()
	db, err := storage.Open(path)
	if err != nil {
		return fmt.Errorf("open database %q: %w", path, err)
	}
	defer db.Close()
	if err := storage.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate database %q: %w", path, err)
	}
	service := workflow.NewService(storage.NewRepository(db), runfiles.New(c.runsRoot))
	service.Now = c.now
	service.NewID = c.newID
	return fn(service)
}

func (c command) flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseRunArgs(args []string) (string, error) {
	var runID string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if value, ok := strings.CutPrefix(arg, "--run="); ok {
			runID = value
			continue
		}
		if arg == "--run" {
			i++
			if i >= len(args) {
				return "", errors.New("--run requires a value")
			}
			runID = args[i]
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return "", fmt.Errorf("unknown flag %q", arg)
		}
		if runID != "" {
			return "", fmt.Errorf("unexpected argument %q", arg)
		}
		runID = arg
	}
	return runID, nil
}

func parseReviseArgs(args []string) (string, string, error) {
	var runID string
	var instruction string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if value, ok := strings.CutPrefix(arg, "--run="); ok {
			runID = value
			continue
		}
		if arg == "--run" {
			i++
			if i >= len(args) {
				return "", "", errors.New("--run requires a value")
			}
			runID = args[i]
			continue
		}
		if value, ok := strings.CutPrefix(arg, "--instruction="); ok {
			instruction = value
			continue
		}
		if arg == "--instruction" {
			i++
			if i >= len(args) {
				return "", "", errors.New("--instruction requires a value")
			}
			instruction = args[i]
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return "", "", fmt.Errorf("unknown flag %q", arg)
		}
		if runID != "" {
			return "", "", fmt.Errorf("unexpected argument %q", arg)
		}
		runID = arg
	}
	return runID, instruction, nil
}

func (c command) printUsage() {
	fmt.Fprintln(c.stdout, `Hermeneia content workflow CLI

Usage:
  hermeneia init              initialize the SQLite database
  hermeneia create            create a content run
  hermeneia list              list content runs
  hermeneia show              show a content run
  hermeneia revise            create a new brief revision
  hermeneia render            render/export run artifacts

Help:
  hermeneia help
  hermeneia --help

Configuration:
  HERMENEIA_DATABASE_PATH  SQLite path (default: data/hermeneia.db)

Examples:
  hermeneia create --topic "AI agents in marketing" --type carousel
  hermeneia revise <run-id> --instruction "Make the hook sharper"
  hermeneia render <run-id>`)
}
