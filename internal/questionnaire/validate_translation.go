package questionnaire

import "slices"

// languages are the tags of the enum language in its order (docs/i18n/SPEC.md §3 "Tags"). A questionnaire's
// original language and its translations take any of them.
var languages = []string{"bs", "hr", "sr-Latn", "sr-Cyrl", "en", "de", "ar", "tr", "it", "fr"}

func knownLanguage(tag string) bool { return slices.Contains(languages, tag) }

// byLanguage orders translations as the enum does.
func byLanguage(a, b Translation) int {
	return slices.Index(languages, a.Language) - slices.Index(languages, b.Language)
}

// flatten is every item of the tree in document order.
func flatten(items []Item) []Item {
	var out []Item
	for _, it := range items {
		out = append(out, it)
		out = append(out, flatten(it.Items)...)
	}
	return out
}

// validateTranslations checks a draft's translations ts against its original language and items: the original
// a known language; each translation a known language other than the original, once, its title and
// description within the original's limits; each item's texts within the original's limits, help text only
// where the original has it, and a choice item's options as many as it has (or none yet), different. An
// incomplete translation passes: publishing refuses it. It returns an *InvalidError naming the offending
// items in document order, or nil.
func validateTranslations(original string, items []Item, ts []Translation) error {
	whole := !knownLanguage(original)
	seen := map[string]bool{}
	for _, t := range ts {
		if !knownLanguage(t.Language) || t.Language == original || seen[t.Language] ||
			chars(t.Title) > maxTitle || chars(t.Description) > maxDescription {
			whole = true
		}
		seen[t.Language] = true
	}
	var bad []string
	for _, it := range flatten(items) {
		for _, t := range ts {
			if e, ok := t.Items[it.LinkID]; ok && !validItemTranslation(it, e) {
				bad = append(bad, it.LinkID)
				break
			}
		}
	}
	if whole || len(bad) > 0 {
		return &InvalidError{LinkIDs: bad}
	}
	return nil
}

func validItemTranslation(it Item, e ItemTranslation) bool {
	if chars(e.Text) > maxText || chars(e.Help) > maxHelp || (e.Help != "" && it.Help == "") {
		return false
	}
	if len(e.Options) == 0 {
		return true
	}
	if len(e.Options) != len(it.Options) {
		return false
	}
	seen := make(map[string]bool, len(e.Options))
	for _, o := range e.Options {
		if chars(o) > maxOption || (!blank(o) && seen[o]) {
			return false
		}
		seen[o] = true
	}
	return true
}

// incomplete is what the translation t lacks of the version q: whether its title or description (when q has
// one), and the items whose text, help text or options it lacks, in document order.
func incomplete(q Questionnaire, t Translation) (whole bool, linkIDs []string) {
	whole = blank(t.Title) || (!blank(q.Description) && blank(t.Description))
	for _, it := range flatten(q.Items) {
		e, ok := t.Items[it.LinkID]
		if !ok || blank(e.Text) || (it.Help != "" && blank(e.Help)) || !allOptions(it, e) {
			linkIDs = append(linkIDs, it.LinkID)
		}
	}
	return whole, linkIDs
}

// isComplete tells whether t has every text of q.
func isComplete(q Questionnaire, t Translation) bool {
	whole, linkIDs := incomplete(q, t)
	return !whole && len(linkIDs) == 0
}

func allOptions(it Item, e ItemTranslation) bool {
	if len(e.Options) != len(it.Options) {
		return false
	}
	return !slices.ContainsFunc(e.Options, blank)
}

// freshen is the translations a draft's save stores: ts (those sent, or the stored ones kept) without the
// items that are no longer among items, and without each text whose original wording changed from old where
// the save sends it as it was stored (it was not translated again), so no one reads the translation of an
// older wording (docs/i18n/SPEC.md §3 "Translations": texts of items that changed are missing until
// translated). They come in the enum's order.
func freshen(old, items []Item, stored, ts []Translation) []Translation {
	before := map[string]Item{}
	for _, it := range flatten(old) {
		before[it.LinkID] = it
	}
	out := make([]Translation, 0, len(ts))
	for _, t := range ts {
		var prev Translation
		if i := slices.IndexFunc(stored, func(s Translation) bool { return s.Language == t.Language }); i >= 0 {
			prev = stored[i]
		}
		kept := make(map[string]ItemTranslation, len(t.Items))
		for _, it := range flatten(items) {
			e, ok := t.Items[it.LinkID]
			if was, existed := before[it.LinkID]; ok && existed {
				e = unchanged(was, it, e, prev.Items[it.LinkID])
			}
			if e.Text != "" || e.Help != "" || len(e.Options) > 0 {
				kept[it.LinkID] = e
			}
		}
		t.Items = kept
		out = append(out, t)
	}
	slices.SortStableFunc(out, byLanguage)
	return out
}

// unchanged is e without each text whose original changed from was to now and that e holds as stored.
func unchanged(was, now Item, e, stored ItemTranslation) ItemTranslation {
	if was.Text != now.Text && e.Text == stored.Text {
		e.Text = ""
	}
	if was.Help != now.Help && e.Help == stored.Help {
		e.Help = ""
	}
	if !slices.Equal(was.Options, now.Options) && slices.Equal(e.Options, stored.Options) {
		e.Options = nil
	}
	return e
}
