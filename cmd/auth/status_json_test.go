package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee-cli/cmd"
)

func TestRunStatusJSONAuthenticated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"login":"andershsueh","name":"Anders","public_repos":3,"public_gists":0,"followers":1,"following":2}`))
	}))
	defer ts.Close()

	var buf bytes.Buffer
	statusCmd.SetOut(&buf)
	statusCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{
		Token:    "secret-token-value",
		APIURL:   ts.URL,
		Timeout:  5,
		Output:   "json",
		Hostname: "gitee.com",
	}))
	if err := runStatus(statusCmd, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "secret-token-value") {
		t.Fatalf("json status leaked the token:\n%s", out)
	}
	var payload statusPayload
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out)
	}
	if !payload.Authenticated || payload.User == nil || payload.User.Login != "andershsueh" {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Token == nil || !strings.Contains(payload.Token.Masked, "...") {
		t.Fatalf("token = %+v", payload.Token)
	}
}

func TestRunStatusJSONMissingToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GITEE_TOKEN", "")
	var buf bytes.Buffer
	statusCmd.SetOut(&buf)
	statusCmd.SetContext(cmd.SetContext(context.Background(), &cmd.Runtime{
		Output:   "json",
		Hostname: "gitee.com",
		Timeout:  5,
	}))
	err := runStatus(statusCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(buf.String(), `"authenticated": false`) {
		t.Fatalf("stdout = %s", buf.String())
	}
}
