// Command salusctl is the operator command (docs/accounts/SPEC.md "Operator commands"). It connects as the
// schema owner (DATABASE_URL), like migrate; every change it makes is audited with salusctl as the agent.
//
//	salusctl create-clinic -name <name> -admin-email <e-mail>   the clinic and its first admin's invite
//	salusctl reset-mfa <account-id>                             a staff member who lost their device
//
// It prints opaque ids only, never an e-mail.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

const usage = "usage: salusctl create-clinic -name <name> -admin-email <e-mail> | reset-mfa <account-id>"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "salusctl:", err)
		os.Exit(1)
	}
}

// services are what the commands call, on the owner's connection.
type services struct {
	tx       *postgres.Transactor
	clinics  *organization.Service
	accounts *auth.Service
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	jobs, err := worker.NewInserter(pool)
	if err != nil {
		return err
	}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	s := services{
		tx:      tx,
		clinics: organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{}),
		accounts: auth.NewService(auth.Ports{
			Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Jobs: auth.NewRiverJobs(jobs),
		}),
	}
	switch args[0] {
	case "create-clinic":
		return s.createClinic(ctx, args[1:], stdout)
	case "reset-mfa":
		if len(args) != 2 {
			return errors.New(usage)
		}
		if err := s.accounts.ResetMFA(ctx, args[1]); err != nil {
			return err
		}
		_, err := fmt.Fprintln(stdout, "reset MFA of account", args[1]+"; its sessions are revoked")
		return err
	default:
		return fmt.Errorf("unknown command %q; %s", args[0], usage)
	}
}

// createClinic creates the clinic and its first admin's invite in one transaction; the notifier sends the
// invite e-mail.
func (s services) createClinic(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("create-clinic", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	name := flags.String("name", "", "the clinic's name")
	email := flags.String("admin-email", "", "the first admin's e-mail")
	if err := flags.Parse(args); err != nil || *name == "" || *email == "" {
		return errors.New(usage)
	}
	var clinic organization.Organization
	var inv auth.Invite
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if clinic, err = s.clinics.Create(ctx, *name); err != nil {
			return err
		}
		inv, err = s.accounts.InviteAdmin(ctx, clinic.ID, *email)
		return err
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "organization %s\ninvite %s\n", clinic.ID, inv.ID)
	return err
}
