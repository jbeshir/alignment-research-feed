package mysql

import (
	"context"
	"database/sql"
	"fmt"
)

import _ "github.com/go-sql-driver/mysql"

func Connect(ctx context.Context, uri string) (*sql.DB, error) {
	db, err := sql.Open("mysql", uri)
	if err != nil {
		return nil, fmt.Errorf("connecting to MySQL DB: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking MySQL DB connection: %w", err)
	}

	return db, nil
}
