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
