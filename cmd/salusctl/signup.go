package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

const signupUsage = `usage: salusctl signup list
       salusctl signup approve <organization-id>`

// signupCommand lists the clinics that signed themselves up and verified their e-mail, and approves one: the clinic is
// then listed in the public directory (docs/clinic-cms, O1, W3). It prints opaque ids and dates only, never an e-mail
// or a clinic's name. The listing is audited as salusctl.
func (s services) signupCommand(ctx context.Context, args []string, stdout io.Writer) error {
	switch {
	case len(args) == 1 && args[0] == "list":
		list, err := s.signups.Signups(ctx)
		if err != nil {
			return err
		}
		for _, sg := range list {
			state := "waiting"
			if !sg.ApprovedAt.IsZero() {
				state = "approved " + sg.ApprovedAt.UTC().Format(time.DateOnly)
			}
			if _, err := fmt.Fprintf(stdout, "organization %s verified %s %s\n", sg.OrganizationID, sg.VerifiedAt.UTC().Format(time.DateOnly), state); err != nil {
				return err
			}
		}
		return nil
	case len(args) == 2 && args[0] == "approve":
		if err := s.signups.Approve(ctx, args[1]); err != nil {
			return err
		}
		_, err := fmt.Fprintln(stdout, "organization", args[1], "listed")
		return err
	default:
		return errors.New(signupUsage)
	}
}
