package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const subscriptionUsage = `usage: salusctl subscription plan -code <code> -name <name> -price-minor <per seat per month> [-currency BAM] [-features a,b] [-inactive]
       salusctl subscription start-trial <organization-id>
       salusctl subscription mark-paid <invoice-number>
       salusctl subscription status <organization-id> active|past_due|suspended|cancelled`

// subscriptionCommand runs the owner's operations on the clinics' membership (docs/plans/clinic-cms.md, package D), on
// the owner's connection: the app may neither edit a plan nor mark an invoice paid. A status moves only forward; a
// payment returns a clinic that was past due or suspended to active. Every move is audited as salusctl. It prints
// opaque ids, numbers and statuses only.
func (s services) subscriptionCommand(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New(subscriptionUsage)
	}
	switch {
	case args[0] == "plan":
		return s.subscriptionPlan(ctx, args[1:], stdout)
	case args[0] == "start-trial" && len(args) == 2:
		sub, err := s.members.StartTrial(ctx, args[1])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "organization %s: %s on plan %s\n", sub.OrganizationID, sub.Status, sub.PlanCode)
		return err
	case args[0] == "mark-paid" && len(args) == 2:
		number, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || number < 1 {
			return errors.New(subscriptionUsage)
		}
		inv, err := s.members.MarkPaid(ctx, number, clock.System{}.Now())
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "invoice %d of organization %s: paid\n", inv.Number, inv.OrganizationID)
		return err
	case args[0] == "status" && len(args) == 3:
		sub, err := s.members.SetStatus(ctx, args[1], subscription.Status(args[2]))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "organization %s: %s\n", sub.OrganizationID, sub.Status)
		return err
	}
	return errors.New(subscriptionUsage)
}

// subscriptionPlan stores or replaces a plan: the prices are the owner's (Q2), so no plan is seeded.
func (s services) subscriptionPlan(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("subscription plan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	code := flags.String("code", "", "the plan's code")
	name := flags.String("name", "", "the plan's name")
	price := flags.Int64("price-minor", -1, "the price per active practitioner per month, in the currency's minor unit")
	currency := flags.String("currency", "BAM", "ISO 4217")
	features := flags.String("features", "", "comma-separated")
	inactive := flags.Bool("inactive", false, "no new clinic gets the plan")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || *price < 0 {
		return errors.New(subscriptionUsage)
	}
	p := subscription.Plan{Code: *code, Name: *name, Currency: *currency, PricePerSeatMinor: *price, Active: !*inactive}
	if *features != "" {
		p.Features = strings.Split(*features, ",")
	}
	if err := s.members.PutPlan(ctx, p); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "plan %s stored\n", p.Code)
	return err
}
