package websocket

import (
	"database/sql"
	"errors"
	"gin_starter/pkg/db/database"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestCommitFailureDoesNotReplay(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("connection reset"))
	r := &chatRepository{db: &database.DB{DB: db}}
	calls := 0
	err = r.runTxWithRetry(func(*sql.Tx) error { calls++; return nil })
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
