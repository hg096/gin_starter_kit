package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestWithTx_CommitsOnSuccess(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	db := &DB{DB: sqlDB}
	mock.ExpectBegin()
	mock.ExpectCommit()

	if err := db.WithTx(func(tx *sql.Tx) error {
		return nil
	}); err != nil {
		t.Fatalf("WithTx returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestWithTx_RollsBackOnError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	db := &DB{DB: sqlDB}
	wantErr := errors.New("work failed")
	mock.ExpectBegin()
	mock.ExpectRollback()

	err = db.WithTx(func(tx *sql.Tx) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WithTx error = %v, want %v", err, wantErr)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
