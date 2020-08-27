package db_test

import (
	"context"
	"errors"

	"github.com/smartatransit/feedback/db"
	"github.com/smartatransit/feedback/db/dbfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB", func() {
	var (
		database *dbfakes.FakeDBDriver
		migrator *dbfakes.FakeMigrator

		dbIface *db.DBDriver

		client db.Client
	)

	BeforeEach(func() {
		database = &dbfakes.FakeDBDriver{}
		migrator = &dbfakes.FakeMigrator{}

		var databaseTmp db.DBDriver = database
		dbIface = &databaseTmp
	})

	JustBeforeEach(func() {
		client = db.New(*dbIface, migrator)
	})

	Describe("Migrate", func() {
		var callErr error
		JustBeforeEach(func() {
			callErr = client.Migrate(context.Background())
		})

		When("the migration fails", func() {
			BeforeEach(func() {
				migrator.UpReturns(errors.New("migration failed"))
			})
			It("returns an error", func() {
				Expect(callErr).To(MatchError("failed migrating database: migration failed"))
			})
		})
		When("all goes well", func() {
			It("succeeds", func() {
				Expect(callErr).To(BeNil())
			})
		})
	})

	Describe("SaveFeedback", func() {
		var callErr error
		JustBeforeEach(func() {
			callErr = client.SaveFeedback(context.Background(), db.Feedback{})
		})

		When("it fails", func() {
			BeforeEach(func() {
				database.ExecContextReturns(nil, errors.New("insert failed"))
			})
			It("returns an error", func() {
				Expect(callErr).To(MatchError("failed saving feedback: insert failed"))
			})
		})
		When("all goes well", func() {
			It("succeeds", func() {
				Expect(callErr).To(BeNil())
			})
		})
	})
})
