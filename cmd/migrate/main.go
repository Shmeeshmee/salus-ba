// Command migrate applies the embedded goose migrations: migrate up | status | create [-dir migrations] <name>.
// It connects as the schema owner. After up, it sets the password of the application role from
// APP_DATABASE_PASSWORD, and of the retention job's role from RETENTION_DATABASE_PASSWORD, each when set.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"syscall"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

const usage = "usage: migrate up | status | create [-dir migrations] <name>"

// The roles the migrations create: appRole, which the application binaries log in as, and retentionRole,
// which only the retention_apply job logs in as.
const (
	appRole       = "salus_app"
	retentionRole = "salus_retention"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "up", "status":
		return withProvider(ctx, func(p *goose.Provider, db postgres.Querier, cfg config.Config) error {
			if args[0] == "up" {
				passwords := [][2]string{{appRole, cfg.AppDatabasePassword}, {retentionRole, config.LoadRetention().Password}}
				return up(ctx, p, db, passwords, stdout)
			}
			return status(ctx, p, stdout)
		})
	case "create":
		return create(args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q; %s", args[0], usage)
	}
}

func withProvider(ctx context.Context, fn func(*goose.Provider, postgres.Querier, config.Config) error) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return err
	}
	return fn(p, pool, cfg)
}

// up applies the migrations, then sets each {role, password} whose password is not empty.
func up(ctx context.Context, p *goose.Provider, db postgres.Querier, passwords [][2]string, stdout io.Writer) error {
	results, err := p.Up(ctx)
	for _, r := range results {
		if _, werr := fmt.Fprintf(stdout, "applied  %s  %s\n", filepath.Base(r.Source.Path), r.Duration); werr != nil {
			return werr
		}
	}
	if err != nil {
		return err
	}
	for _, rp := range passwords {
		if rp[1] == "" {
			continue
		}
		if err := postgres.SetPassword(ctx, db, rp[0], rp[1]); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(stdout, "set the password of", rp[0]); err != nil {
			return err
		}
	}
	return nil
}

func status(ctx context.Context, p *goose.Provider, stdout io.Writer) error {
	statuses, err := p.Status(ctx)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		if _, err := fmt.Fprintf(stdout, "%-8s %s\n", s.State, filepath.Base(s.Source.Path)); err != nil {
			return err
		}
	}
	return nil
}

var (
	migrationName = regexp.MustCompile(`^[a-z0-9_]+$`)
	migrationFile = regexp.MustCompile(`^(\d{5})_[a-z0-9_]+\.sql$`)
)

// create writes the next NNNNN_name.sql file in dir. goose's own Create numbers by timestamp unless a
// package-level switch is flipped, so the numbering is done here.
func create(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("create", flag.ContinueOnError)
	dir := flags.String("dir", "migrations", "directory holding the migrations")
	if err := flags.Parse(args); err != nil {
		return err
	}
	name := flags.Arg(0)
	if flags.NArg() != 1 || !migrationName.MatchString(name) {
		return errors.New("create needs one name of lowercase letters, digits and underscores")
	}
	entries, err := os.ReadDir(*dir)
	if err != nil {
		return err
	}
	var numbers []int
	for _, e := range entries {
		if m := migrationFile.FindStringSubmatch(e.Name()); m != nil {
			n, _ := strconv.Atoi(m[1]) // five digits, always a number
			numbers = append(numbers, n)
		}
	}
	next := 1
	if len(numbers) > 0 {
		next = slices.Max(numbers) + 1
	}
	path := filepath.Join(*dir, fmt.Sprintf("%05d_%s.sql", next, name))
	body := "-- +goose Up\n-- TODO: say what this migration does.\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "created", path)
	return err
}
