package main

import (
	"database/sql"
	"log"
	"net/http"

	flags "github.com/jessevdk/go-flags"
	_ "github.com/lib/pq" //provides the postgres driver for database/sql

	"github.com/smartatransit/feedback/api"
	"github.com/smartatransit/feedback/db"
)

var opts struct {
	PostgresConnectionString  string `long:"postgres-connection-string" env:"POSTGRES_CONNECTION_STRING" required:"true"`
	MigrationsPath            string `long:"migrations-path" env:"MIGRATIONS_PATH" default:"/db-migrations/"`
	OutageReportAlertTTLHours int    `long:"outage-report-alert-ttl-hours" env:"OUTAGE_REPORT_ALERT_TTL_HOURS" default:"48"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatal(err)
	}

	pg, err := sql.Open("postgres", opts.PostgresConnectionString)
	if err != nil {
		log.Fatal(err)
	}

	dbClient := db.New(opts.MigrationsPath, opts.PostgresConnectionString, pg)
	apiClient := api.New(dbClient)

	srv := http.NewServeMux()
	srv.HandleFunc("/v1/feedback", apiClient.SaveFeedback)
	srv.HandleFunc("/v1/health", apiClient.Health)

	// TODO log
	http.ListenAndServe(":8080", srv)
}
