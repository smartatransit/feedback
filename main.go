package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4"
	flags "github.com/jessevdk/go-flags"
	_ "github.com/lib/pq" //provides the postgres driver for database/sql
	"github.com/sirupsen/logrus"

	"github.com/smartatransit/feedback/api"
	"github.com/smartatransit/feedback/db"
)

var opts struct {
	PostgresURL               string `long:"postgres-url" env:"POSTGRES_URL" required:"true"`
	MigrationsPath            string `long:"migrations-path" env:"MIGRATIONS_PATH" default:"/db-migrations/"`
	OutageReportAlertTTLHours int    `long:"outage-report-alert-ttl-hours" env:"OUTAGE_REPORT_ALERT_TTL_HOURS" default:"48"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatal(err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	database, err := sql.Open("postgres", opts.PostgresURL)
	if err != nil {
		logger.Error(err.Error())
		log.Fatal()
	}

	migrator, err := migrate.New("file", "file://"+opts.MigrationsPath)
	if err != nil {
		logger.Error(err.Error())
		log.Fatal()
	}

	dbClient := db.New(database, migrator)

	apiClient := api.New(logger, dbClient)

	err = dbClient.Migrate(context.Background())
	if err != nil && !strings.Contains(err.Error(), "no change") {
		logger.Error(err.Error())
		log.Fatal()
	}

	srv := http.NewServeMux()
	srv.HandleFunc("/v1/feedback", apiClient.SaveFeedback)
	srv.HandleFunc("/v1/health", apiClient.Health)

	logger.Info("Starting API...")
	_ = http.ListenAndServe(":8080", srv)
}
