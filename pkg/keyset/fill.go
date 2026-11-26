package keyset

// Fill reads one page of at most count matches in batches, for a list some of whose items are not matches (those
// the caller may not see). read gives at most limit items after *after in the list's order (from where the page
// starts when after is nil); keep gives the matches among a batch, in order and without side effects (it may see
// items past the page's end). Each batch is count+1 items, the last only to tell whether the list goes on; Fill
// reads on past the items it left out until the page is full or the list ends. more tells that an item follows
// the page's last match (it may itself be left out, so the next page may come back empty).
func Fill[T, M any](count int, read func(after *T, limit int) ([]T, error), keep func(batch []T) ([]M, error)) (page []M, more bool, err error) {
	var after *T
	for {
		batch, err := read(after, count+1)
		if err != nil {
			return nil, false, err
		}
		goesOn := len(batch) > count
		if goesOn {
			batch = batch[:count]
		}
		kept, err := keep(batch)
		if err != nil {
			return nil, false, err
		}
		page = append(page, kept...)
		switch {
		case len(page) > count:
			return page[:count], true, nil
		case len(page) == count, !goesOn:
			return page, goesOn, nil
		}
		after = &batch[len(batch)-1]
	}
}
