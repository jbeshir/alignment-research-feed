package mysql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

const driverParamStr string = "?parseTime=true"

func Connect(ctx context.Context, uri string) (*sql.DB, error) {
	db, err := sql.Open("mysql", uri+driverParamStr)
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
