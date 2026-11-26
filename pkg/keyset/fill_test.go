package keyset_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/keyset"
)

// list reads the items after *after (from the first when nil), limit at most, as a repository reads a sorted
// table, and keeps the limits it was asked for.
type list struct {
	items  []int
	limits []int
}

func (l *list) read(after *int, limit int) ([]int, error) {
	l.limits = append(l.limits, limit)
	var out []int
	for _, it := range l.items {
		if (after == nil || it > *after) && len(out) < limit {
			out = append(out, it)
		}
	}
	return out, nil
}

// even keeps the even items, doubled, as a service keeps those the caller may see in the form it gives them.
func even(batch []int) ([]int, error) {
	var out []int
	for _, it := range batch {
		if it%2 == 0 {
			out = append(out, it*2)
		}
	}
	return out, nil
}

func TestFill_ReadsBatchesUntilThePageIsFullAndTellsWhetherMoreMayFollow(t *testing.T) {
	for _, tt := range []struct {
		name   string
		items  []int
		want   []int
		limits []int
		count  int
		more   bool
	}{
		{"a page of the first matches, more after", []int{2, 4, 6, 8}, []int{4, 8}, []int{3}, 2, true},
		{"the last page, exactly full", []int{2, 4}, []int{4, 8}, []int{3}, 2, false},
		{"the last page, short", []int{2}, []int{4}, []int{3}, 2, false},
		{"none", nil, nil, []int{3}, 2, false},
		{"the batch's left-out items read past, in other batches", []int{1, 2, 3, 5, 7, 8, 10}, []int{4, 16}, []int{3, 3, 3}, 2, true},
		{"left-out items to the end", []int{1, 2, 3, 5, 7}, []int{4}, []int{3, 3, 3}, 2, false},
		{"a batch with more matches than the page lacks", []int{1, 2, 3, 4, 6, 8}, []int{4, 8, 12}, []int{4, 4}, 3, true},
		{"the item after a full page is left out: more may follow", []int{2, 4, 5}, []int{4, 8}, []int{3}, 2, true},
	} {
		l := &list{items: tt.items}

		got, more, err := keyset.Fill(tt.count, l.read, even)

		if err != nil || !slices.Equal(got, tt.want) || more != tt.more || !slices.Equal(l.limits, tt.limits) {
			t.Errorf("%s: Fill = %v, %v, %v after reads of %v; want %v, %v after %v", tt.name, got, more, err, l.limits, tt.want, tt.more, tt.limits)
		}
	}
}

func TestFill_ConcatenatedPagesAreTheWholeAnswer(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}
	whole, _ := even(items)
	for count := 1; count <= 10; count++ {
		var all []int
		var start *int // where the next link says the page starts: after the previous page's last match
		for pages := 0; pages <= len(items); pages++ {
			l, first := &list{items: items}, start
			page, more, err := keyset.Fill(count, func(after *int, limit int) ([]int, error) {
				if after == nil {
					after = first
				}
				return l.read(after, limit)
			}, even)
			if err != nil {
				t.Fatal(err)
			}
			all = append(all, page...)
			if !more {
				break
			}
			last := page[len(page)-1] / 2
			start = &last
		}
		if !slices.Equal(all, whole) {
			t.Errorf("pages of %d = %v, want %v", count, all, whole)
		}
	}
}

func TestFill_StopsOnAnError(t *testing.T) {
	boom := errors.New("boom")
	if _, _, err := keyset.Fill(2, func(*int, int) ([]int, error) { return nil, boom }, even); !errors.Is(err, boom) {
		t.Errorf("a read's error = %v, want %v", err, boom)
	}
	keep := func([]int) ([]int, error) { return nil, boom }
	if _, _, err := keyset.Fill(2, (&list{items: []int{1}}).read, keep); !errors.Is(err, boom) {
		t.Errorf("a keep's error = %v, want %v", err, boom)
	}
}
