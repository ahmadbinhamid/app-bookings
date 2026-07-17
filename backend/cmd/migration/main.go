package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"app-booking/database/migrator"
	"app-booking/internal/config"
	"app-booking/internal/db"

	// Imported for its side effect: each migration file's init() registers
	// itself with the migrator. The migrations package is otherwise unused here.
	_ "app-booking/database/migrations"
)

type Command string

const (
	Create Command = "create"
	Fresh  Command = "fresh"
	Run    Command = "run"
)

const dir = "database/migrations"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "no command given (create <name> | run | fresh)")
		os.Exit(1)
	}
	cmd := Command(os.Args[1])

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "could not create folder:", err)
		os.Exit(1)
	}

	switch cmd {
	case Create:
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "create needs a table name")
			os.Exit(1)
		}
		create(os.Args[2])
	case Fresh:
		fresh()
	case Run:
		run()
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cmd)
		os.Exit(1)
	}
}

func create(tableName string) {
	ts := time.Now().Format("20060102150405") // generate ONCE
	name := ts + "_" + tableName
	filename := name + ".go"

	file, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create migration file:", err)
		os.Exit(1)
	}
	defer file.Close()

	// The init() makes the file register itself with the runner on import.
	content := fmt.Sprintf(`package migrations

import (
	"database/sql"

	"app-booking/database/migrator"
)

func init() {
	migrator.Register(migrator.Migration{
		Name: %q,
		Up:   Up_%s,
		Down: Down_%s,
	})
}

func Up_%s(db *sql.DB) error {
	_, err := db.Exec()
	return err
}

func Down_%s(db *sql.DB) error {
	_, err := db.Exec()
	return err
}
`, name, ts, ts, ts, ts)

	if _, err := file.WriteString(content); err != nil {
		fmt.Fprintln(os.Stderr, "could not write migration file:", err)
		os.Exit(1)
	}
	fmt.Println("created:", filepath.Join(dir, filename))
}

func run() {
	conn := connect()
	defer conn.Close()

	if err := migrator.Run(conn); err != nil {
		fmt.Fprintln(os.Stderr, "migration failed:", err)
		os.Exit(1)
	}
}

func fresh() {
	conn := connect()
	defer conn.Close()

	if err := migrator.Fresh(conn); err != nil {
		fmt.Fprintln(os.Stderr, "fresh failed:", err)
		os.Exit(1)
	}
}

func connect() *sql.DB {
	conn, err := db.Connect(config.Load())
	if err != nil {
		fmt.Fprintln(os.Stderr, "db connection failed:", err)
		os.Exit(1)
	}
	return conn
}
