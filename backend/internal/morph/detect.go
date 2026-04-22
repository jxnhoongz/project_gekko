package morph

import (
	"sort"
	"strings"

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// GeneRow is the minimal view of a gecko_genes+genetic_dictionary join row
// needed for morph detection. Callers convert DB-specific rows into this type.
type GeneRow struct {
	TraitID         int32
	TraitName       string
	InheritanceType db.InheritanceType
	Zygosity        db.Zygosity
	SuperFormName   string
}

// ComboRequirement describes one trait required by a combo.
type ComboRequirement struct {
	TraitID          int32
	RequiredZygosity db.Zygosity
}

// Combo describes one named morph combination.
type Combo struct {
	Name         string
	Requirements []ComboRequirement
}

// zygosityRank lets us compare zygosity: HOM(2) > HET(1) > POSS_HET(0).
var zygosityRank = map[db.Zygosity]int{
	db.ZygosityHOM:     2,
	db.ZygosityHET:     1,
	db.ZygosityPOSSHET: 0,
}

// Detect labels a gecko from its gene rows and the full combo catalog.
// Combos are sorted longest-requirements-first so larger combos win over
// their subsets (Diablo Blanco beats Raptor+leftover Blizzard).
// Once a combo matches, its traits are covered and cannot be claimed by
// a shorter combo.
func Detect(genes []GeneRow, combos []Combo) string {
	// Sort combos: longest requirements first.
	sorted := make([]Combo, len(combos))
	copy(sorted, combos)
	sort.SliceStable(sorted, func(i, j int) bool {
		return len(sorted[i].Requirements) > len(sorted[j].Requirements)
	})

	// Fast lookup: traitID → zygosity.
	zygByTrait := make(map[int32]db.Zygosity, len(genes))
	for _, g := range genes {
		zygByTrait[g.TraitID] = g.Zygosity
	}

	var matchedNames []string
	covered := make(map[int32]bool)

	for _, combo := range sorted {
		// Skip if any required trait is already covered by a longer combo.
		skip := false
		for _, req := range combo.Requirements {
			if covered[req.TraitID] {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Check all requirements satisfied.
		ok := true
		for _, req := range combo.Requirements {
			actual, has := zygByTrait[req.TraitID]
			if !has || zygosityRank[actual] < zygosityRank[req.RequiredZygosity] {
				ok = false
				break
			}
		}
		if ok {
			matchedNames = append(matchedNames, combo.Name)
			for _, req := range combo.Requirements {
				covered[req.TraitID] = true
			}
		}
	}

	// Remaining uncovered traits.
	var remaining []string
	for _, g := range genes {
		if covered[g.TraitID] {
			continue
		}
		switch {
		case g.InheritanceType == db.InheritanceTypeCODOMINANT && g.Zygosity == db.ZygosityHOM:
			if g.SuperFormName != "" {
				remaining = append(remaining, g.SuperFormName)
			} else {
				remaining = append(remaining, g.TraitName)
			}
		case g.Zygosity == db.ZygosityHET:
			remaining = append(remaining, "het "+g.TraitName)
		case g.Zygosity == db.ZygosityPOSSHET:
			remaining = append(remaining, "poss. het "+g.TraitName)
		default:
			remaining = append(remaining, g.TraitName)
		}
	}

	parts := append(matchedNames, remaining...)
	if len(parts) == 0 {
		return "Normal"
	}
	return strings.Join(parts, " ")
}
