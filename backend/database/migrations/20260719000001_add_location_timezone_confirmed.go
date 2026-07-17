package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: "20260719000001_add_location_timezone_confirmed",
		Up:   Up_20260719000001,
		Down: Down_20260719000001,
	})
}

// timezone_confirmed distinguishes "an admin explicitly set this location's
// timezone" from "this is just the UTC placeholder sync left behind." A
// manual value always wins and is never overwritten by sync — see
// internal/modules/location/repository.go's Upsert.
func Up_20260719000001(db *sql.DB) error {
	_, err := db.Exec(`
		ALTER TABLE locations
		ADD COLUMN timezone_confirmed BOOLEAN NOT NULL DEFAULT FALSE AFTER timezone
	`)
	return err
}

func Down_20260719000001(db *sql.DB) error {
	_, err := db.Exec(`ALTER TABLE locations DROP COLUMN timezone_confirmed`)
	return err
}
