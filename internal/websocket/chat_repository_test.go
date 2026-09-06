package websocket

import (
	stderrors "errors"
	"gin_starter/pkg/db/database"
	appErrors "gin_starter/pkg/errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestIsRetryableDBConnError(t *testing.T) {
	retryable := []error{
		stderrors.New("driver: bad connection"),
		stderrors.New("read tcp: connection reset by peer"),
		appErrors.Wrap(stderrors.New("invalid connection"), "DATABASE_ERROR", "wrapped db error"),
	}
	for _, err := range retryable {
		if !isRetryableDBConnError(err) {
			t.Fatalf("expected retryable error, got: %v", err)
		}
	}

	nonRetryable := []error{
		stderrors.New("duplicate entry"),
		appErrors.New("BAD_REQUEST", "invalid payload"),
	}
	for _, err := range nonRetryable {
		if isRetryableDBConnError(err) {
			t.Fatalf("expected non-retryable error, got: %v", err)
		}
	}
}

func TestEnsureDefaultAdminRoomReadsUsersBeforeUpsertingMembers(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}
	repo := &chatRepository{
		db:   db,
		base: database.NewRepository(db),
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE _chat_rooms SET")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), AdminLoungeRoomKey).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT u_id
			FROM _user
			WHERE u_auth_type IN ('TA', 'A', 'M', 'G', 'AG')
		`)).
		WillReturnRows(sqlmock.NewRows([]string{"u_id"}).
			AddRow("qwe11").
			AddRow("admin2").
			CloseError(nil))
	for _, userID := range []string{"qwe11", "admin2"} {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO _chat_room_members (crm_joined_at, crm_room_key, crm_user_id)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE crm_joined_at = crm_joined_at`)).
			WithArgs(sqlmock.AnyArg(), AdminLoungeRoomKey, userID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	if err := repo.EnsureDefaultAdminRoom(); err != nil {
		t.Fatalf("EnsureDefaultAdminRoom returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
