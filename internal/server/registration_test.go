package server

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAddMutualFriendship(t *testing.T) {
	a, mock := testApp(t)
	query := regexp.QuoteMeta("INSERT INTO `yu_friend` (`create_time`,`create_user`,`friend_user_id`,`status`) VALUES (?,?,?,?)")
	mock.ExpectExec(query).
		WithArgs(int64(100), int64(2), int64(9), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(query).
		WithArgs(int64(100), int64(9), int64(2), 1).
		WillReturnResult(sqlmock.NewResult(2, 1))

	if e := a.addMutualFriendship(context.Background(), a.db, 2, 9, 100); e != nil {
		t.Fatal(e)
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}

func TestAddMutualFriendshipStopsOnFailure(t *testing.T) {
	a, mock := testApp(t)
	query := regexp.QuoteMeta("INSERT INTO `yu_friend` (`create_time`,`create_user`,`friend_user_id`,`status`) VALUES (?,?,?,?)")
	want := errors.New("friend insert failed")
	mock.ExpectExec(query).
		WithArgs(int64(100), int64(2), int64(9), 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(query).
		WithArgs(int64(100), int64(9), int64(2), 1).
		WillReturnError(want)

	if e := a.addMutualFriendship(context.Background(), a.db, 2, 9, 100); !errors.Is(e, want) {
		t.Fatalf("error = %v, want %v", e, want)
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}
