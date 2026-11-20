package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/web/locale"
)

func hostRequest(t *testing.T, user *model.User, method, path, body string) map[string]any {
	t.Helper()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("LOGIN_USER_ROW", user)
		c.Set("base_path", "/")
		c.Set("I18n", func(kind locale.I18nType, key string, params ...string) string { return key })
		c.Next()
	})
	NewHostController(r.Group("/panel/api/hosts"))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("host HTTP %d: %s", w.Code, w.Body.String())
	}
	return response
}

func TestHostsHTTPRejectsForeignGroupsAndResellerWrites(t *testing.T) {
	f := newIdorFixture(t)
	h := &model.Host{GroupId: "victim-group", InboundId: f.aliInbound.Id, Remark: "private-edge", Address: "private.example", Port: 443, Security: "same"}
	if err := database.GetDB().Create(h).Error; err != nil {
		t.Fatal(err)
	}
	response := hostRequest(t, f.reza, http.MethodGet, "/panel/api/hosts/list", "")
	if response["success"] != true || len(response["obj"].([]any)) != 0 {
		t.Fatalf("foreign hosts leaked: %#v", response)
	}
	for _, request := range []struct{ path, body string }{
		{"/panel/api/hosts/del/victim-group", ""},
		{"/panel/api/hosts/setEnable/victim-group", `{"enable":false}`},
		{"/panel/api/hosts/reorder", `{"ids":["victim-group"]}`},
		{"/panel/api/hosts/add", `{"remark":"foreign","inboundIds":[` + strings.TrimSpace(string(mustHostJSON(t, f.aliInbound.Id))) + `],"hosts":["evil.example"]}`},
	} {
		if got := hostRequest(t, f.reza, http.MethodPost, request.path, request.body); got["success"] != false {
			t.Fatalf("cross-admin request succeeded: %s %#v", request.path, got)
		}
	}
	reseller := &model.User{Id: f.reza.Id, IsReseller: true, Enable: true, Permissions: f.reza.Permissions}
	if got := hostRequest(t, reseller, http.MethodPost, "/panel/api/hosts/del/victim-group", ""); got["success"] != false {
		t.Fatalf("reseller changed shared Host policy: %#v", got)
	}
	var kept model.Host
	if err := database.GetDB().First(&kept, h.Id).Error; err != nil {
		t.Fatal(err)
	}
	if kept.Remark != "private-edge" || kept.IsDisabled {
		t.Fatalf("refused requests changed host: %#v", kept)
	}
}

func mustHostJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestHostsHTTPCreateUpdateClearSNIAndBulkDelete(t *testing.T) {
	f := newIdorFixture(t)
	request := map[string]any{"remark": "managed", "inboundIds": []int{f.rezaInbound.Id}, "hosts": []string{"cdn.example:8443"}, "snis": []string{"one.sni"}, "security": "same"}
	response := hostRequest(t, f.reza, http.MethodPost, "/panel/api/hosts/add", string(mustHostJSON(t, request)))
	if response["success"] != true {
		t.Fatalf("create failed: %#v", response)
	}
	rows := response["obj"].([]any)
	if len(rows) != 1 {
		t.Fatalf("create rows: %#v", rows)
	}
	row := rows[0].(map[string]any)
	group := row["groupId"].(string)
	id := row["id"]
	request["snis"] = []string{}
	request["sni"] = ""
	request["remark"] = "updated"
	response = hostRequest(t, f.reza, http.MethodPost, "/panel/api/hosts/update/"+group, string(mustHostJSON(t, request)))
	if response["success"] != true {
		t.Fatalf("update failed: %#v", response)
	}
	updated := response["obj"].([]any)[0].(map[string]any)
	if updated["id"] != id || updated["sni"] != "" || updated["remark"] != "updated" {
		t.Fatalf("update lost identity or could not clear SNI: %#v", updated)
	}
	response = hostRequest(t, f.reza, http.MethodPost, "/panel/api/hosts/bulk/del", string(mustHostJSON(t, map[string]any{"ids": []string{group}})))
	if response["success"] != true {
		t.Fatalf("delete failed: %#v", response)
	}
	response = hostRequest(t, f.reza, http.MethodGet, "/panel/api/hosts/raw", "")
	if response["success"] != true || len(response["obj"].([]any)) != 0 {
		t.Fatalf("deleted hosts remain: %#v", response)
	}
}
