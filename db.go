package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB é nil quando o portal roda sem banco (modo local: progresso só no navegador, sem login).
var DB *sql.DB

func conectaBanco(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)
	// o Postgres do docker compose pode demorar alguns segundos para aceitar conexões
	var ultimo error
	for i := 0; i < 30; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		ultimo = db.PingContext(ctx)
		cancel()
		if ultimo == nil {
			return db, nil
		}
		time.Sleep(time.Second)
	}
	db.Close()
	return nil, fmt.Errorf("banco não respondeu: %w", ultimo)
}

// migra aplica, em ordem e uma única vez, os arquivos migrations/NNN_nome.sql embutidos no binário.
// Um advisory lock impede que duas instâncias migrem ao mesmo tempo.
func migra(db *sql.DB) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(827027)`); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `SELECT pg_advisory_unlock(827027)`)
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		versao TEXT PRIMARY KEY, aplicada_em TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	arquivos, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(arquivos)
	for _, f := range arquivos {
		versao := strings.TrimSuffix(strings.TrimPrefix(f, "migrations/"), ".sql")
		var existe bool
		if err := conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE versao=$1)`, versao).Scan(&existe); err != nil {
			return err
		}
		if existe {
			continue
		}
		sqlTxt, err := fs.ReadFile(migrationsFS, f)
		if err != nil {
			return err
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(sqlTxt)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", versao, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(versao) VALUES($1)`, versao); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("migration aplicada: %s", versao)
	}
	return nil
}
