package database

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCount_AllRows_WhenWhereEmpty(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	repo := NewRepository(&DB{DB: sqlDB})

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM _blog")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := repo.Count("_blog", "")
	if err != nil {
		t.Fatalf("Count returned error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestExec_AllowsOnlyValidatedDML(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	repo := NewRepository(&DB{DB: sqlDB})

	mock.ExpectExec(regexp.QuoteMeta("UPDATE _user SET u_name = ? WHERE u_id = ?")).
		WithArgs("tester", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := repo.Exec("UPDATE _user SET u_name = ? WHERE u_id = ?", "tester", "u1"); err != nil {
		t.Fatalf("Exec returned error for valid DML: %v", err)
	}

	if _, err := repo.Exec("UPDATE _user SET u_name = ?", "tester"); err == nil {
		t.Fatalf("expected UPDATE without WHERE to be rejected")
	}
	if _, err := repo.Exec("ALTER TABLE _user ADD COLUMN x INT"); err == nil {
		t.Fatalf("expected schema SQL to be rejected by Exec")
	}
	if _, err := repo.Exec("DELETE FROM _user; DELETE FROM _blog"); err == nil {
		t.Fatalf("expected multiple statements to be rejected")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestExecSchema_AllowsOnlySchemaSQL(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	repo := NewRepository(&DB{DB: sqlDB})

	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS _sample (id INT);")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if _, err := repo.ExecSchema("CREATE TABLE IF NOT EXISTS _sample (id INT);"); err != nil {
		t.Fatalf("ExecSchema returned error for schema SQL: %v", err)
	}
	if _, err := repo.ExecSchema("INSERT INTO _sample (id) VALUES (?)", 1); err == nil {
		t.Fatalf("expected DML to be rejected by ExecSchema")
	}
	if _, err := repo.ExecSchema("DROP TABLE _sample"); err == nil {
		t.Fatalf("expected DROP to be rejected by ExecSchema")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRepositoryWithTxUsesSameMethodNames(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	repo := NewRepository(&DB{DB: sqlDB})

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO _user")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE _user SET")).
		WithArgs(sqlmock.AnyArg(), "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM _user WHERE u_id = ?")).
		WithArgs("u1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.WithTx(func(q *TxRepository) error {
		if _, err := q.Insert("_user", map[string]interface{}{"u_id": "u1", "u_name": "tester"}); err != nil {
			return err
		}
		if _, err := q.Update("_user", map[string]interface{}{"u_name": "updated"}, "u_id = ?", "u1"); err != nil {
			return err
		}
		if _, err := q.Delete("_user", "u_id = ?", "u1"); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
