package main

import (
	"context"
	nethttp "net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/config"
	apihttp "github.com/jxnhoongz/project_gekko/backend/internal/http"
)

func main() {
	_ = godotenv.Load(".env.local")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config load failed")
	}

	if cfg.LogLevel == "debug" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen})
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db connection failed")
	}
	defer pool.Close()

	signer := auth.NewJWTSigner(cfg.JWTSecret, time.Duration(cfg.JWTTTLHours)*time.Hour)

	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Logger, chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w nethttp.ResponseWriter, req *nethttp.Request) {
		if err := pool.Ping(req.Context()); err != nil {
			nethttp.Error(w, `{"status":"db down"}`, nethttp.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	fs := nethttp.StripPrefix("/uploads/", nethttp.FileServer(nethttp.Dir(cfg.UploadDir)))
	r.Handle("/uploads/*", fs)

	apihttp.MountAuth(r, pool, signer)
	apihttp.MountWaitlist(r, pool, signer)
	apihttp.MountSchema(r, pool, signer)
	apihttp.MountGeckos(r, pool, signer)
	apihttp.MountMedia(r, pool, signer, cfg)

	log.Info().Msgf("listening on :%s", cfg.Port)
	if err := nethttp.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal().Err(err).Send()
	}
}
