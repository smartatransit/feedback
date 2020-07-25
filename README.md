Feedback
========

A simple go microservice for managing user feedback.

The database schema is managed using the [golang-migrate](https://github.com/golang-migrate/migrate/) tool. The schema can be updated by adding a pair of SQL files to the `db-migrations/` directory, and some guidelines can be found [here](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md).
