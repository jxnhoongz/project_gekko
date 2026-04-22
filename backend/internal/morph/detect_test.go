package morph_test

import (
	"testing"

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
	"github.com/jxnhoongz/project_gekko/backend/internal/morph"
)

const (
	idTremper   = int32(1)
	idEclipse   = int32(2)
	idBlizzard  = int32(3)
	idMackSnow  = int32(4)
	idRainwater = int32(5)
	idBell      = int32(6)
)

var testCombos = []morph.Combo{
	{
		Name: "Diablo Blanco",
		Requirements: []morph.ComboRequirement{
			{TraitID: idTremper, RequiredZygosity: db.ZygosityHOM},
			{TraitID: idBlizzard, RequiredZygosity: db.ZygosityHOM},
			{TraitID: idEclipse, RequiredZygosity: db.ZygosityHOM},
		},
	},
	{
		Name: "Raptor",
		Requirements: []morph.ComboRequirement{
			{TraitID: idTremper, RequiredZygosity: db.ZygosityHOM},
			{TraitID: idEclipse, RequiredZygosity: db.ZygosityHOM},
		},
	},
	{
		Name: "Firewater",
		Requirements: []morph.ComboRequirement{
			{TraitID: idRainwater, RequiredZygosity: db.ZygosityHOM},
			{TraitID: idEclipse, RequiredZygosity: db.ZygosityHOM},
		},
	},
}

func g(id int32, name string, itype db.InheritanceType, zyg db.Zygosity, super string) morph.GeneRow {
	return morph.GeneRow{TraitID: id, TraitName: name, InheritanceType: itype, Zygosity: zyg, SuperFormName: super}
}

func TestDetect_NoTraits_ReturnsNormal(t *testing.T) {
	if got := morph.Detect(nil, testCombos); got != "Normal" {
		t.Fatalf("want Normal, got %q", got)
	}
}

func TestDetect_Raptor(t *testing.T) {
	genes := []morph.GeneRow{
		g(idTremper, "Tremper Albino", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idEclipse, "Eclipse", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
	}
	if got := morph.Detect(genes, testCombos); got != "Raptor" {
		t.Fatalf("want Raptor, got %q", got)
	}
}

// Critical: longest-match-first must pick Diablo Blanco over Raptor + leftover Blizzard.
func TestDetect_DiabloBlanco_NotRaptorBlizzard(t *testing.T) {
	genes := []morph.GeneRow{
		g(idTremper, "Tremper Albino", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idEclipse, "Eclipse", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idBlizzard, "Blizzard", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
	}
	if got := morph.Detect(genes, testCombos); got != "Diablo Blanco" {
		t.Fatalf("want 'Diablo Blanco', got %q", got)
	}
}

func TestDetect_CoDominantHOM_UsesSuperFormName(t *testing.T) {
	genes := []morph.GeneRow{
		g(idMackSnow, "Mack Snow", db.InheritanceTypeCODOMINANT, db.ZygosityHOM, "Super Snow"),
	}
	if got := morph.Detect(genes, testCombos); got != "Super Snow" {
		t.Fatalf("want 'Super Snow', got %q", got)
	}
}

func TestDetect_RaptorHetMackSnow(t *testing.T) {
	genes := []morph.GeneRow{
		g(idTremper, "Tremper Albino", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idEclipse, "Eclipse", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idMackSnow, "Mack Snow", db.InheritanceTypeCODOMINANT, db.ZygosityHET, "Super Snow"),
	}
	if got := morph.Detect(genes, testCombos); got != "Raptor het Mack Snow" {
		t.Fatalf("want 'Raptor het Mack Snow', got %q", got)
	}
}

func TestDetect_PossHet(t *testing.T) {
	genes := []morph.GeneRow{
		g(idTremper, "Tremper Albino", db.InheritanceTypeRECESSIVE, db.ZygosityPOSSHET, ""),
	}
	if got := morph.Detect(genes, testCombos); got != "poss. het Tremper Albino" {
		t.Fatalf("want 'poss. het Tremper Albino', got %q", got)
	}
}

func TestDetect_Polygenic_HOM_UsesTraitName(t *testing.T) {
	genes := []morph.GeneRow{
		g(7, "Tangerine", db.InheritanceTypePOLYGENIC, db.ZygosityHOM, ""),
	}
	if got := morph.Detect(genes, testCombos); got != "Tangerine" {
		t.Fatalf("want 'Tangerine', got %q", got)
	}
}
