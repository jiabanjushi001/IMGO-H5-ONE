package server

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestTaskStatusValuesMatchAdminUI(t *testing.T) {
	a, mock := testApp(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/manage/task/getTaskList", nil)
	for _, state := range []struct{ enabled, status string }{{"true", "active"}, {"false", "stop"}} {
		mock.ExpectQuery("SELECT value FROM `yu_config` WHERE name='maintenance'").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"enabled":` + state.enabled + `}`))
		mock.ExpectQuery("SELECT value FROM `yu_config` WHERE name='chatInfo'").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{}`))
		v, e := a.task(&request{c: c})
		if e != nil {
			t.Fatal(e)
		}
		list := v.([]M)
		if list[0]["status"] != state.status || list[1]["status"] != "active" {
			t.Fatal(list)
		}
	}
	if e := mock.ExpectationsWereMet(); e != nil {
		t.Fatal(e)
	}
}
