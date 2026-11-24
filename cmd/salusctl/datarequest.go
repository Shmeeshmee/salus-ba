package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/Shmeeshmee/salus/internal/datarequest"
)

const dataRequestUsage = `usage: salusctl datarequest list
       salusctl datarequest show <request-id>
       salusctl datarequest answer <request-id> <version> done|done_partly_kept|identity_unverified|manifestly_unfounded|excessive|retention_law|no_data
       salusctl datarequest extend <request-id> <version> complex|many_requests
       salusctl datarequest reopen-account <account-id>`

// dataRequestCommand runs the operator's handling of the data subject's requests to the Salus account
// (docs/clinic-cms/WAVE5.md S1, D5) on the owner's connection: list, show with history, answer, extend once while not
// overdue, and reopen a closed account before its erasure. Every one is audited as salusctl. It prints opaque ids,
// codes and dates only.
func (s services) dataRequestCommand(ctx context.Context, args []string, stdout io.Writer) error {
	switch {
	case len(args) == 1 && args[0] == "list":
		all, err := s.operator.Requests(ctx)
		if err != nil {
			return err
		}
		for _, r := range all {
			if _, err := fmt.Fprintln(stdout, requestLine(r)); err != nil {
				return err
			}
		}
		return nil
	case len(args) == 2 && args[0] == "show":
		r, history, err := s.operator.Request(ctx, args[1])
		if err != nil {
			return err
		}
		return printRequest(stdout, r, history)
	case len(args) == 4 && args[0] == "answer":
		version, err := strconv.Atoi(args[2])
		if err != nil {
			return errors.New(dataRequestUsage)
		}
		code := datarequest.AnswerCode(args[3])
		status := datarequest.StatusRefused
		if code == datarequest.AnswerDone || code == datarequest.AnswerDonePartlyKept {
			status = datarequest.StatusFulfilled
		}
		r, err := s.operator.Answer(ctx, args[1], datarequest.Answer{Status: status, Code: code, Version: version})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, requestLine(r))
		return err
	case len(args) == 4 && args[0] == "extend":
		version, err := strconv.Atoi(args[2])
		if err != nil {
			return errors.New(dataRequestUsage)
		}
		r, err := s.operator.Extend(ctx, args[1], datarequest.ExtensionReason(args[3]), version)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, requestLine(r))
		return err
	case len(args) == 2 && args[0] == "reopen-account":
		r, err := s.operator.ReopenAccount(ctx, args[1])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "account %s reopened; erasure request %s withdrawn\n", args[1], r.ID)
		return err
	}
	return errors.New(dataRequestUsage)
}

// requestLine is one request on a line: its id, kind, status, deadline in the operator's time zone, version and
// account.
func requestLine(r datarequest.Request) string {
	line := fmt.Sprintf("%s %s %s due %s version %d account %s", r.ID, r.Kind, r.Status, operatorDate(r.Deadline()), r.Version, r.AccountID)
	if r.AnswerCode != "" {
		line += " answer " + string(r.AnswerCode)
	}
	if r.ExtensionReason != "" {
		line += " extended " + string(r.ExtensionReason)
	}
	return line
}

// printRequest prints the request and its history, oldest first.
func printRequest(w io.Writer, r datarequest.Request, history []datarequest.Change) error {
	if _, err := fmt.Fprintf(w, "%s\nreceived %s\ndue %s\n", requestLine(r), operatorDate(r.ReceivedAt), operatorDate(r.DueAt)); err != nil {
		return err
	}
	for _, c := range history {
		by := c.ChangedBy
		if by == "" {
			by = "salusctl or a job"
		}
		if _, err := fmt.Fprintf(w, "history %s %s by %s\n", operatorDate(c.ChangedAt), c.Status, by); err != nil {
			return err
		}
	}
	return nil
}

// operatorDate writes t in the operator's time zone, to the minute.
func operatorDate(t time.Time) string {
	loc, err := time.LoadLocation(datarequest.OperatorTimeZone)
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04 MST")
}
