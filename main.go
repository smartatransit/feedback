package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
	_ "github.com/lib/pq" //provides the postgres driver for database/sql
	"github.com/sirupsen/logrus"

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

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	pg, err := sql.Open("postgres", opts.PostgresConnectionString)
	if err != nil {
		logger.Error(err.Error())
		log.Fatal()
	}

	dbClient := db.New(opts.MigrationsPath, opts.PostgresConnectionString, pg)
	apiClient := api.New(logger, dbClient)

	err = dbClient.Migrate(context.Background())
	if err != nil && !strings.Contains(err.Error(), "no change") {
		logger.Error(err.Error())
		log.Fatal()
	}

	srv := http.NewServeMux()
	srv.HandleFunc("/v1/feedback", apiClient.SaveFeedback)
	srv.HandleFunc("/v1/health", apiClient.Health)

	logger.Info(err.Error())
	fmt.Println("Starting API...")
	http.ListenAndServe(":8080", srv)
}
