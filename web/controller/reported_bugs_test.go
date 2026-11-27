package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/web/locale"
	"github.com/mhsanaei/3x-ui/v2/web/service"
)

func reportedBugRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("base_path", "/")
		c.Set("I18n", func(_ locale.I18nType, key string, _ ...string) string { return key })
		c.Next()
	})
	return router
}

func TestNodeAddReadsJSONTokenAndRejectsMissingToken(t *testing.T) {
	newPermTestDB(t)
	wantToken := "remote-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantToken {
			http.Error(w, "bad authorization: "+got, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"test","inbounds":2}`))
	}))
	defer server.Close()
	host, portText, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	router := reportedBugRouter()
	router.POST("/nodes/add", (&NodeController{}).add)
	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/nodes/add", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}

	missing := post(fmt.Sprintf(`{"name":"missing-token","address":%q,"port":%d}`, host, port))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing token status = %d body=%s", missing.Code, missing.Body.String())
	}
	var count int64
	if err := database.GetDB().Model(&model.Node{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("missing token created a node: count=%d err=%v", count, err)
	}

	body := fmt.Sprintf(`{"name":"json-child","scheme":"http","address":%q,"port":%d,"basePath":"/","token":%q,"syncMode":"all","enabled":true}`, host, port, wantToken)
	response := post(body)
	if response.Code != http.StatusOK {
		t.Fatalf("JSON node add status = %d body=%s", response.Code, response.Body.String())
	}
	var msg struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &msg); err != nil || !msg.Success {
		t.Fatalf("JSON node add failed: %s err=%v", response.Body.String(), err)
	}
	if strings.Contains(response.Body.String(), wantToken) {
		t.Fatal("node API token leaked in the add response")
	}
	node, err := (&service.NodeService{}).Get(1)
	if err != nil || node.Token != wantToken || node.Status != "online" {
		t.Fatalf("stored node did not retain/use JSON token: %+v err=%v", node, err)
	}
}

func TestCreateApiTokenRejectsUnknownScopeWithBadRequest(t *testing.T) {
	newPermTestDB(t)
	router := reportedBugRouter()
	router.POST("/nodes/tokens", (&NodeController{}).createToken)
	req := httptest.NewRequest(http.MethodPost, "/nodes/tokens", strings.NewReader(url.Values{
		"name": {"bad-scope"}, "scopes": {"node:sync"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown scope status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var count int64
	if err := database.GetDB().Model(&model.ApiToken{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("invalid scope persisted a token: count=%d err=%v", count, err)
	}
}

func TestUpdateSettingsMergesPartialJSONAndFormBodies(t *testing.T) {
	newPermTestDB(t)
	settings := &service.SettingService{}
	baseline, err := settings.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	baseline.WebPort = 2083
	baseline.SubPort = 2097
	baseline.SubTitle = "before"
	if err := settings.UpdateAllSetting(baseline); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	router := reportedBugRouter()
	router.POST("/setting/update", (&SettingController{settingService: *settings}).updateSetting)
	post := func(contentType, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/setting/update", strings.NewReader(body))
		req.Header.Set("Content-Type", contentType)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	jsonResponse := post("application/json", `{"tgBotEnable":true}`)
	if jsonResponse.Code != http.StatusOK || !strings.Contains(jsonResponse.Body.String(), `"success":true`) {
		t.Fatalf("partial JSON update failed: %d %s", jsonResponse.Code, jsonResponse.Body.String())
	}
	formResponse := post("application/x-www-form-urlencoded", "subTitle=after")
	if formResponse.Code != http.StatusOK || !strings.Contains(formResponse.Body.String(), `"success":true`) {
		t.Fatalf("partial form update failed: %d %s", formResponse.Code, formResponse.Body.String())
	}
	after, err := settings.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	if after.WebPort != 2083 || after.SubPort != 2097 || after.SubTitle != "after" || !after.TgBotEnable {
		t.Fatalf("partial updates lost existing settings: %+v", after)
	}
}
