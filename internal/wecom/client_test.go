package wecom

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestReadAPIResponseReturnsHelpfulErrorForEmptyTodoBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	_, err := readAPIResponse(resp, "/cgi-bin/todo/add")
	if err == nil {
		t.Fatal("readAPIResponse returned nil error, want helpful error")
	}
	if !strings.Contains(err.Error(), "empty response body") {
		t.Fatalf("error = %q, want empty response body detail", err.Error())
	}
	if !strings.Contains(err.Error(), "/cgi-bin/todo/add") {
		t.Fatalf("error = %q, want endpoint path", err.Error())
	}
}

func TestReadAPIResponseReturnsAPIErrorMessage(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader(`{"errcode":48002,"errmsg":"api forbidden"}`)),
	}

	_, err := readAPIResponse(resp, "/cgi-bin/user/list_id")
	if err == nil {
		t.Fatal("readAPIResponse returned nil error, want api error")
	}
	if !strings.Contains(err.Error(), "api forbidden") {
		t.Fatalf("error = %q, want api message", err.Error())
	}
}
