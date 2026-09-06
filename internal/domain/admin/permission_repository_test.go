package admin

import (
	"gin_starter/pkg/db/database"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSplitFilterCSV(t *testing.T) {
	raw := " admin.account.read,admin.account.read, ,admin.stats.read "
	got := splitFilterCSV(raw)
	want := []string{"admin.account.read", "admin.stats.read"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected split result: got=%v want=%v", got, want)
	}
}

func TestEnsureAdminPageColumnsSkipsExistingColumnsAndIndex(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}
	repo := &permissionRepository{
		db:   db,
		base: database.NewRepository(db),
	}

	columnQuery := regexp.QuoteMeta(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = '_a_admin_pages'
		  AND COLUMN_NAME = ?
	`)
	for _, column := range []string{"group_key", "group_label", "group_order", "visible_roles"} {
		mock.ExpectQuery(columnQuery).
			WithArgs(column).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
	}

	indexQuery := regexp.QuoteMeta(`
		SELECT COUNT(*)
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = '_a_admin_pages'
		  AND INDEX_NAME = ?
	`)
	mock.ExpectQuery(indexQuery).
		WithArgs("idx_a_admin_pages_enabled_group_sort").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))

	if err := repo.ensureAdminPageColumns(); err != nil {
		t.Fatalf("ensureAdminPageColumns returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
