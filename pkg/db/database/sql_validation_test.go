package database

import "testing"

func TestDMLWhereMustBeTopLevel(t *testing.T) {
	for _, query := range []string{
		"DELETE FROM _blog /* where */",
		"UPDATE _blog SET title = 'some where text'",
		"UPDATE _blog SET title = (SELECT title FROM other WHERE id = 1)",
		"DELETE FROM _blog -- where",
		"DELETE FROM _blog WHERE id=1; DELETE FROM _blog",
	} {
		if err := validateDMLQuery(query); err == nil {
			t.Errorf("accepted unsafe SQL: %s", query)
		}
	}
	for _, query := range []string{
		"UPDATE _blog SET title=? WHERE id=?",
		"UPDATE _blog SET title='a;b' WHERE id=1;",
		"DELETE FROM _blog WHERE id IN (SELECT id FROM other WHERE enabled=1)",
	} {
		if err := validateDMLQuery(query); err != nil {
			t.Errorf("rejected SQL: %s: %v", query, err)
		}
	}
}
