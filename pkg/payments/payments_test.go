package payments_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/payments"
)

func TestCheck_TakesAKeyAPositiveAmountACurrencyAndACustomer(t *testing.T) {
	t.Parallel()
	ok := payments.Charge{Key: "k", Currency: "EUR", Customer: "c", AmountMinor: 1}
	if err := payments.Check(ok); err != nil {
		t.Errorf("a valid charge: %v", err)
	}
	for name, change := range map[string]func(*payments.Charge){
		"no key": func(c *payments.Charge) { c.Key = "" }, "negative": func(c *payments.Charge) { c.AmountMinor = -1 },
		"lowercase": func(c *payments.Charge) { c.Currency = "eur" }, "two letters": func(c *payments.Charge) { c.Currency = "EU" },
		"no customer": func(c *payments.Charge) { c.Customer = "" },
	} {
		c := ok
		change(&c)
		if err := payments.Check(c); !errors.Is(err, payments.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
