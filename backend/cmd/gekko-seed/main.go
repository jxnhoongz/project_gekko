package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func main() {
	_ = godotenv.Load(".env.local")

	email := flag.String("email", "", "admin email (required)")
	password := flag.String("password", "", "admin password, >=8 chars (required)")
	name := flag.String("name", "", "display name (optional)")
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "usage: gekko-seed -email <email> -password <pw> [-name <name>]")
		os.Exit(2)
	}

	hash, err := auth.HashPassword(*password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "password error: %v\n", err)
		os.Exit(1)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DB_URL is required (set in backend/.env.local)")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	q := db.New(pool)

	existing, err := q.GetAdminByEmail(ctx, *email)
	if err == nil {
		fmt.Printf("admin already exists: id=%d email=%s\n", existing.ID, existing.Email)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		fmt.Fprintf(os.Stderr, "lookup admin: %v\n", err)
		os.Exit(1)
	}

	created, err := q.CreateAdmin(ctx, db.CreateAdminParams{
		Email:        *email,
		PasswordHash: hash,
		Name:         nullText(name),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create admin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("created admin id=%d email=%s\n", created.ID, created.Email)
}

func nullText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
