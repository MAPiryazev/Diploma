package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wb-go/wbf/dbpg"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/config"
)

func Connect(cfg *config.Config) (*dbpg.DB, error) {
	masterDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	opts := &dbpg.Options{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db, err := dbpg.New(masterDSN, nil, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Master.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func RunMigrations(db *dbpg.DB, migrationDir ...string) error {
	dir := "migrations"
	if len(migrationDir) > 0 && migrationDir[0] != "" {
		dir = migrationDir[0]
	}

	ctx := context.Background()

	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	applied, err := loadAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}

		if applied[name] {
			continue
		}

		files = append(files, filepath.Join(dir, name))
	}

	sort.Strings(files)

	for _, path := range files {
		if err := applyMigrationFile(ctx, db, path); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, db *dbpg.DB) error {
	const q = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`
	if _, err := db.Master.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

func loadAppliedMigrations(ctx context.Context, db *dbpg.DB) (map[string]bool, error) {
	const q = `SELECT filename FROM schema_migrations`

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("select schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool, 16)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schema_migrations rows: %w", err)
	}

	return applied, nil
}

func applyMigrationFile(ctx context.Context, db *dbpg.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("migration file not found %s: %w", path, err)
		}
		return fmt.Errorf("read migration file %s: %w", path, err)
	}

	stmts := splitSQLStatements(string(content))
	if len(stmts) == 0 {
		return nil
	}

	tx, err := db.Master.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration %s error: %w", filepath.Base(path), err)
		}
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(filename) VALUES ($1)`, filepath.Base(path)); err != nil {
		return fmt.Errorf("insert schema_migrations %s: %w", filepath.Base(path), err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", filepath.Base(path), err)
	}

	return nil
}

func splitSQLStatements(sqlText string) []string {
	var res []string

	var b strings.Builder
	inSingle := false

	flush := func() {
		s := strings.TrimSpace(b.String())
		b.Reset()
		if s != "" {
			res = append(res, s)
		}
	}

	for i := 0; i < len(sqlText); i++ {
		ch := sqlText[i]

		if ch == '\'' {
			if inSingle && i+1 < len(sqlText) && sqlText[i+1] == '\'' {
				b.WriteByte(ch)
				b.WriteByte(sqlText[i+1])
				i++
				continue
			}
			inSingle = !inSingle
			b.WriteByte(ch)
			continue
		}

		if ch == ';' && !inSingle {
			flush()
			continue
		}

		b.WriteByte(ch)
	}

	flush()
	return res
}
