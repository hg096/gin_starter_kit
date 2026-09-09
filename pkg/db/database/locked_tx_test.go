package database

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestNamedLockSurroundsTransaction(t *testing.T) {
	for _, fail := range []bool{false, true} {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectQuery("SELECT GET_LOCK").WithArgs("bootstrap").WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(1))
		mock.ExpectBegin()
		if fail {
			mock.ExpectRollback()
		} else {
			mock.ExpectCommit()
		}
		mock.ExpectQuery("SELECT RELEASE_LOCK").WithArgs("bootstrap").WillReturnRows(sqlmock.NewRows([]string{"released"}).AddRow(1))
		err = (&DB{DB: db}).WithNamedLockTx(context.Background(), "bootstrap", func(*sql.Tx) error {
			if fail {
				return errors.New("failed")
			}
			return nil
		})
		if (err != nil) != fail {
			t.Fatalf("err=%v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		db.Close()
	}
}
