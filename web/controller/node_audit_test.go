package controller

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v2/web/service"
)

func TestNodeTokenAcceptsBrowserFormAndJSON(t *testing.T) {
	newPermTestDB(t)
	for _, request := range []struct{ contentType, body string }{
		{"application/x-www-form-urlencoded", "name=browser&scopes=node-sync%2Cnode-admin"},
		{"application/json", `{"name":"json","scopes":"node-sync,node-admin"}`},
	} {
		router := gin.New()
		router.POST("/panel/nodes/tokens", (&NodeController{}).createToken)
		req := httptest.NewRequest("POST", "/panel/nodes/tokens", strings.NewReader(request.body))
		req.Header.Set("Content-Type", request.contentType)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		var response struct {
			Success bool   `json:"success"`
			Msg     string `json:"msg"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if !response.Success {
			t.Fatalf("%s: %s", request.contentType, recorder.Body.String())
		}
		token, err := (&service.ApiTokenService{}).Verify(response.Msg)
		if err != nil || !(&service.ApiTokenService{}).HasScope(token, service.ScopeNodeAdmin) {
			t.Fatalf("browser token did not authenticate with requested scopes: %v", err)
		}
	}
}
