package terminology

import (
	"context"
	"fmt"
	"slices"
)

// Catalog serves the diagnosis groups and the places (00042, docs/plans/analytics.md §3): reference data, not
// audited, no patient data.
type Catalog struct {
	repo CatalogRepository
}

// NewCatalog returns the Catalog on repo.
func NewCatalog(repo CatalogRepository) *Catalog { return &Catalog{repo: repo} }

// GroupEntry is a group as the filters show it: its code, its parent, and its name in one language.
type GroupEntry struct {
	Code         string
	Parent       string
	Name         string
	ReviewStatus string
}

// Groups returns the grouping of ICD-10 codes, without the names, for the aggregate builds (on salus_stats).
func (c *Catalog) Groups(ctx context.Context) (Groups, error) {
	groups, ranges, err := c.repo.Grouping(ctx)
	if err != nil {
		return Groups{}, fmt.Errorf("terminology: diagnosis groups: %w", err)
	}
	return NewGroups(groups, ranges)
}

// GroupNames returns every group in its order with its name in language, one of the ten app languages
// (ErrInvalidSearch otherwise). Staff only: a patient gets ErrForbidden.
func (c *Catalog) GroupNames(ctx context.Context, caller Caller, language string) ([]GroupEntry, error) {
	if !caller.Staff {
		return nil, ErrForbidden
	}
	if language == "la" || !slices.Contains(languages, language) {
		return nil, ErrInvalidSearch
	}
	groups, _, err := c.repo.DiagnosisGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("terminology: diagnosis groups: %w", err)
	}
	out := make([]GroupEntry, len(groups))
	for i, d := range groups {
		out[i] = GroupEntry{Code: d.Code, Parent: d.Parent}
		for _, n := range d.Names {
			if n.Language == language {
				out[i].Name, out[i].ReviewStatus = n.Text, n.Status
			}
		}
	}
	return out, nil
}

// Places returns the places of a clinic's location. Staff only: a patient gets ErrForbidden.
func (c *Catalog) Places(ctx context.Context, caller Caller) ([]Place, error) {
	if !caller.Staff {
		return nil, ErrForbidden
	}
	places, err := c.repo.Places(ctx)
	if err != nil {
		return nil, fmt.Errorf("terminology: places: %w", err)
	}
	return places, nil
}
