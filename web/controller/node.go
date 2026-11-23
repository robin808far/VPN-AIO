package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/config"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/web/service"

	"github.com/gin-gonic/gin"
)

// NodeController serves both halves of the multi-panel feature: the master-side
// Nodes page and its JSON API (session + super admin), and the node-side
// /nodeapi endpoints a remote master calls with a bearer API token.
type NodeController struct {
	BaseController
	nodeService     service.NodeService
	apiTokenService service.ApiTokenService
}

// NewNodeController registers the routes on the top-level base-path group.
func NewNodeController(g *gin.RouterGroup) *NodeController {
	a := &NodeController{}
	a.initRouter(g)
	return a
}

func (a *NodeController) initRouter(g *gin.RouterGroup) {
	// Master side: the Nodes page and its management API.
	panel := g.Group("/panel")
	panel.Use(a.checkLogin)
	panel.GET("/nodes", requireSuperAdmin(), a.page)
	nodes := panel.Group("/nodes")
	nodes.Use(requireSuperAdmin())
	{
		nodes.GET("/list", a.list)
		nodes.POST("/add", a.add)
		nodes.POST("/update", a.update)
		nodes.POST("/del/:id", a.del)
		nodes.GET("/probe/:id", a.probe)
		nodes.GET("/inbounds/:id", a.remoteInbounds)
		// API tokens are managed here too: creating them is what the remote
		// panel's super admin does in the reference (Settings → API Token).
		nodes.GET("/tokens", a.listTokens)
		nodes.POST("/tokens", a.createToken)
		nodes.POST("/tokens/del/:id", a.revokeToken)
	}

	// Node side: what a remote master calls on this panel. Bearer token, no session.
	nodeapi := g.Group("/nodeapi")
	{
		nodeapi.GET("/status", a.requireScope(service.ScopeNodeSync), a.nodeStatus)
		nodeapi.GET("/inbounds", a.requireScope(service.ScopeNodeSync), a.nodeInbounds)
		nodeapi.GET("/traffic", a.requireScope(service.ScopeNodeSync), a.nodeTraffic)
	}
}

// page renders the Nodes management page.
func (a *NodeController) page(c *gin.Context) {
	html(c, "nodes.html", "pages.nodes.title", nil)
}

// list returns all registered nodes.
func (a *NodeController) list(c *gin.Context) {
	nodes, err := a.nodeService.List()
	jsonObj(c, nodes, err)
}

// add registers a new remote panel.
func (a *NodeController) add(c *gin.Context) {
	node := &model.Node{}
	if err := c.ShouldBind(node); err != nil {
		pureJsonMsg(c, http.StatusBadRequest, false, "Invalid node payload")
		return
	}
	node.Id = 0
	created, err := a.nodeService.Add(node)
	jsonMsgObj(c, "Node added", created, err)
}

// update edits an existing remote panel.
func (a *NodeController) update(c *gin.Context) {
	form := &model.Node{}
	if err := c.ShouldBind(form); err != nil {
		pureJsonMsg(c, http.StatusBadRequest, false, "Invalid node payload")
		return
	}
	updated, err := a.nodeService.Update(form.Id, form)
	jsonMsgObj(c, "Node updated", updated, err)
}

// del removes a node registration.
func (a *NodeController) del(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	err := a.nodeService.Del(id)
	jsonMsg(c, "Node deleted", err)
}

// probe re-checks one node immediately.
func (a *NodeController) probe(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	node, err := a.nodeService.Get(id)
	if err != nil {
		jsonMsgObj(c, "Probe failed", nil, err)
		return
	}
	status := a.nodeService.Probe(node)
	jsonMsgObj(c, "Probed", gin.H{"node": node, "status": status}, nil)
}

// remoteInbounds lists the inbounds of a remote panel.
func (a *NodeController) remoteInbounds(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	node, err := a.nodeService.Get(id)
	if err != nil {
		jsonObj(c, nil, err)
		return
	}
	inbounds, err := a.nodeService.RemoteInbounds(node)
	jsonObj(c, inbounds, err)
}

// listTokens returns the API tokens without ever exposing the plaintext.
func (a *NodeController) listTokens(c *gin.Context) {
	tokens, err := a.apiTokenService.List()
	jsonObj(c, tokens, err)
}

// createToken mints a token and returns the plaintext exactly once.
func (a *NodeController) createToken(c *gin.Context) {
	form := struct {
		Name   string `json:"name"`
		Scopes string `json:"scopes"`
	}{}
	if err := c.ShouldBind(&form); err != nil {
		pureJsonMsg(c, http.StatusBadRequest, false, "Invalid token payload")
		return
	}
	plaintext, token, err := a.apiTokenService.Create(form.Name, form.Scopes)
	jsonMsgObj(c, plaintext, token, err)
}

// revokeToken deletes an API token.
func (a *NodeController) revokeToken(c *gin.Context) {
	id, ok := pathInt(c, "id")
	if !ok {
		return
	}
	err := a.apiTokenService.Revoke(id)
	jsonMsg(c, "Token revoked", err)
}

// requireScope enforces bearer-token auth with the given scope on /nodeapi.
func (a *NodeController) requireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token, err := a.apiTokenService.Verify(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if !a.apiTokenService.HasScope(token, scope) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient scope"})
			return
		}
		c.Next()
	}
}

// nodeStatus answers the remote master's probe: version, Xray state and counts.
func (a *NodeController) nodeStatus(c *gin.Context) {
	status := service.NodeStatus{Version: config.GetVersion()}
	status.XrayRunning = (&service.XrayService{}).IsXrayRunning()
	database.GetDB().Table("inbounds").Count(&status.Inbounds)
	database.GetDB().Table("accounts").Count(&status.Clients)
	var total struct{ Traffic int64 }
	database.GetDB().Table("inbounds").Select("COALESCE(SUM(up + down),0) AS traffic").Scan(&total)
	status.Traffic = total.Traffic
	c.JSON(http.StatusOK, status)
}

// nodeInbounds lists this panel's inbounds as a summary for the remote master.
func (a *NodeController) nodeInbounds(c *gin.Context) {
	inbounds := make([]map[string]any, 0)
	database.GetDB().Table("inbounds").
		Select("id, remark, protocol, port, enable, up, down, total, expiry_time AS expiryTime").
		Find(&inbounds)
	c.JSON(http.StatusOK, inbounds)
}

// nodeTraffic reports per-inbound traffic counters.
func (a *NodeController) nodeTraffic(c *gin.Context) {
	traffic := make([]map[string]any, 0)
	database.GetDB().Table("inbounds").
		Select("id, remark, protocol, up, down, total, expiry_time AS expiryTime").
		Find(&traffic)
	c.JSON(http.StatusOK, traffic)
}

// pathInt parses an :id route parameter; on failure it has already replied 400.
func pathInt(c *gin.Context, name string) (int, bool) {
	value, err := strconv.Atoi(c.Param(name))
	if err != nil || value <= 0 {
		pureJsonMsg(c, http.StatusBadRequest, false, "Invalid id")
		return 0, false
	}
	return value, true
}
