// gekko-seed-demo seeds species, common morphs, and the initial 6-gecko
// demo collection. Idempotent: safe to run more than once.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

type speciesSeed struct {
	code           db.SpeciesCode
	commonName     string
	scientificName string
	description    string
}

type traitSeed struct {
	species     db.SpeciesCode
	name        string
	code        string
	dominant    bool
	description string
}

type geckoSeed struct {
	code      string
	name      string
	species   db.SpeciesCode
	sex       db.Sex
	hatchDate string // "2023-06-12"
	status    db.GeckoStatus
	priceUsd  string // "180.00" or "" for null
	notes     string
	traits    []traitRef
}

type traitRef struct {
	name     string
	zygosity db.Zygosity
}

var seedSpecies = []speciesSeed{
	{db.SpeciesCodeLP, "Leopard Gecko", "Eublepharis macularius", "Terrestrial gecko from South Asia; most common captive species."},
	{db.SpeciesCodeCR, "Crested Gecko", "Correlophus ciliatus", "Arboreal New Caledonian gecko; famous for its eyelash crests."},
	{db.SpeciesCodeAF, "African Fat-tailed Gecko", "Hemitheconyx caudicinctus", "West African terrestrial gecko; tail fat reserves."},
}

var seedTraits = []traitSeed{
	// Leopard traits
	{db.SpeciesCodeLP, "Tangerine", "TANG", false, "Orange body tone line."},
	{db.SpeciesCodeLP, "Mack Snow", "MACK", false, "Co-dominant pastel hypomelanism."},
	{db.SpeciesCodeLP, "Tremper Albino", "TREMPER", false, "T-plus albino strain."},
	{db.SpeciesCodeLP, "Bell Albino", "BELL", false, "T-minus albino strain, incompatible with Tremper."},
	{db.SpeciesCodeLP, "Rainwater Albino", "RAINWATER", false, "Third albino strain."},
	{db.SpeciesCodeLP, "Eclipse", "ECLIPSE", false, "Solid black or red eyes."},
	{db.SpeciesCodeLP, "Blizzard", "BLIZZARD", false, "Patternless white."},
	{db.SpeciesCodeLP, "Enigma", "ENIGMA", true, "Dominant jumbled pattern; neuro issues common."},
	{db.SpeciesCodeLP, "Super Snow", "SUPERMACK", false, "Homozygous Mack Snow — eyes go solid black."},
	{db.SpeciesCodeLP, "W&Y", "WY", true, "White & Yellow; dominant."},

	// Crested traits
	{db.SpeciesCodeCR, "Harlequin", "HARLE", false, "Heavy patterned pattern on legs + back."},
	{db.SpeciesCodeCR, "Dalmatian", "DAL", false, "Black spotted pattern."},
	{db.SpeciesCodeCR, "Pinstripe", "PIN", false, "Raised dorsal scales down the stripe."},
	{db.SpeciesCodeCR, "Flame", "FLAME", false, "High-contrast dorsal flame."},
	{db.SpeciesCodeCR, "Lilly White", "LILLY", true, "Co-dominant white pigment spread."},
	{db.SpeciesCodeCR, "Axanthic", "AX", false, "Black & white — no yellow/red pigment."},

	// African Fat-tail traits
	{db.SpeciesCodeAF, "Oreo", "OREO", false, "Co-dominant cream + dark band pattern."},
	{db.SpeciesCodeAF, "Whiteout", "WHITE", false, "Co-dominant whited pattern."},
	{db.SpeciesCodeAF, "Zulu", "ZULU", false, "Tangerine-like line."},
	{db.SpeciesCodeAF, "Caramel Albino", "CAR_ALB", false, "T-plus albino."},
}

var seedGeckos = []geckoSeed{
	{
		code: "ZG-001", name: "Apsara", species: db.SpeciesCodeLP, sex: db.SexF,
		hatchDate: "2023-06-12", status: db.GeckoStatusBREEDING,
		notes: "Proven breeder, calm temperament.",
		traits: []traitRef{{"Tangerine", db.ZygosityHOM}},
	},
	{
		code: "ZG-002", name: "Rithy", species: db.SpeciesCodeLP, sex: db.SexM,
		hatchDate: "2022-09-03", status: db.GeckoStatusBREEDING,
		notes: "Holdback from 2022 clutch.",
		traits: []traitRef{{"Mack Snow", db.ZygosityHET}, {"Tremper Albino", db.ZygosityHET}},
	},
	{
		code: "ZG-003", name: "Chandra", species: db.SpeciesCodeCR, sex: db.SexF,
		hatchDate: "2024-01-20", status: db.GeckoStatusPERSONAL,
		notes: "Pet, not for sale.",
		traits: []traitRef{{"Harlequin", db.ZygosityHOM}},
	},
	{
		code: "ZG-004", name: "Suri", species: db.SpeciesCodeCR, sex: db.SexF,
		hatchDate: "2024-03-08", status: db.GeckoStatusAVAILABLE,
		priceUsd: "180.00",
		notes:    "High-spot dalmatian, good eater.",
		traits:   []traitRef{{"Dalmatian", db.ZygosityHOM}},
	},
	{
		code: "ZG-005", name: "Khmer", species: db.SpeciesCodeAF, sex: db.SexM,
		hatchDate: "2023-11-11", status: db.GeckoStatusBREEDING,
		notes:  "Imported line, strong oreo pattern.",
		traits: []traitRef{{"Oreo", db.ZygosityHOM}},
	},
	{
		code: "ZG-006", name: "Veasna", species: db.SpeciesCodeLP, sex: db.SexF,
		hatchDate: "2024-05-22", status: db.GeckoStatusHOLD,
		priceUsd: "220.00",
		notes:    "On hold for Dara until May 5.",
		traits:   []traitRef{{"Bell Albino", db.ZygosityHOM}},
	},
}

func main() {
	_ = godotenv.Load(".env.local")
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DB_URL is required (set in backend/.env.local)")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		die("db connect", err)
	}
	defer pool.Close()

	q := db.New(pool)

	fmt.Println("seeding species...")
	speciesIDs := map[db.SpeciesCode]int32{}
	for _, s := range seedSpecies {
		row, err := q.GetSpeciesByCode(ctx, s.code)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			die("lookup species "+string(s.code), err)
		}
		if err == nil {
			speciesIDs[s.code] = row.ID
			fmt.Printf("  %s already exists (id=%d)\n", s.code, row.ID)
			continue
		}
		created, err := q.CreateSpecies(ctx, db.CreateSpeciesParams{
			Code:           s.code,
			CommonName:     s.commonName,
			ScientificName: pgtype.Text{String: s.scientificName, Valid: true},
			Description:    pgtype.Text{String: s.description, Valid: true},
		})
		if err != nil {
			die("create species "+string(s.code), err)
		}
		speciesIDs[s.code] = created.ID
		fmt.Printf("  + %s (id=%d)\n", s.code, created.ID)
	}

	fmt.Println("seeding genetic traits...")
	traitIDs := map[string]int32{} // key = "LP:Tangerine"
	for _, t := range seedTraits {
		spID := speciesIDs[t.species]
		row, err := q.GetTraitByNameAndSpecies(ctx, db.GetTraitByNameAndSpeciesParams{
			SpeciesID: spID,
			Lower:     t.name,
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			die("lookup trait "+t.name, err)
		}
		key := string(t.species) + ":" + t.name
		if err == nil {
			traitIDs[key] = row.ID
			continue
		}
		created, err := q.CreateTrait(ctx, db.CreateTraitParams{
			SpeciesID:   spID,
			TraitName:   t.name,
			TraitCode:   pgtype.Text{String: t.code, Valid: t.code != ""},
			Description: pgtype.Text{String: t.description, Valid: t.description != ""},
			Column5:     t.dominant,
		})
		if err != nil {
			die("create trait "+t.name, err)
		}
		traitIDs[key] = created.ID
		fmt.Printf("  + %s/%s (id=%d)\n", t.species, t.name, created.ID)
	}

	fmt.Println("seeding demo geckos...")
	for _, g := range seedGeckos {
		existing, err := q.GetGeckoByCode(ctx, g.code)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			die("lookup gecko "+g.code, err)
		}
		if err == nil {
			fmt.Printf("  %s already exists (id=%d)\n", g.code, existing.ID)
			continue
		}
		hatchedAt, err := time.Parse("2006-01-02", g.hatchDate)
		if err != nil {
			die("parse hatch date "+g.hatchDate, err)
		}
		price := pgtype.Numeric{}
		if g.priceUsd != "" {
			if err := price.Scan(g.priceUsd); err != nil {
				die("parse price "+g.priceUsd, err)
			}
		}
		created, err := q.CreateGecko(ctx, db.CreateGeckoParams{
			Code:         g.code,
			Name:         pgtype.Text{String: g.name, Valid: true},
			SpeciesID:    speciesIDs[g.species],
			Sex:          g.sex,
			HatchDate:    pgtype.Date{Time: hatchedAt, Valid: true},
			AcquiredDate: pgtype.Date{Valid: false},
			Column7:      db.NullGeckoStatus{GeckoStatus: g.status, Valid: true},
			SireID:       pgtype.Int4{Valid: false},
			DamID:        pgtype.Int4{Valid: false},
			ListPriceUsd: price,
			Notes:        pgtype.Text{String: g.notes, Valid: g.notes != ""},
		})
		if err != nil {
			die("create gecko "+g.code, err)
		}
		fmt.Printf("  + %s %q (id=%d)\n", g.code, g.name, created.ID)

		for _, tr := range g.traits {
			key := string(g.species) + ":" + tr.name
			tid, ok := traitIDs[key]
			if !ok {
				fmt.Fprintf(os.Stderr, "  ! trait %q not found for %s, skipping\n", tr.name, g.species)
				continue
			}
			_, err := q.CreateGeckoGene(ctx, db.CreateGeckoGeneParams{
				GeckoID:  created.ID,
				TraitID:  tid,
				Zygosity: tr.zygosity,
			})
			if err != nil {
				die(fmt.Sprintf("assign trait %s to %s", tr.name, g.code), err)
			}
			fmt.Printf("    · %s (%s)\n", tr.name, tr.zygosity)
		}
	}

	fmt.Println("done.")
}

func die(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
	os.Exit(1)
}
