package controller

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/session"
	"github.com/mhsanaei/3x-ui/v2/web/websocket"
)

type HostController struct{ hosts service.HostService }

func requireHostAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil || user.IsReseller {
			deny(c, http.StatusForbidden, "pages.admins.forbidden", landingPath(c))
			return
		}
		c.Next()
	}
}

func NewHostController(g *gin.RouterGroup) *HostController {
	a := &HostController{}
	g.Use(requirePerm(model.PermAccessInbounds), requireHostAdmin())
	g.GET("/list", a.list)
	g.GET("/raw", a.raw)
	g.GET("/get/:groupId", a.get)
	g.GET("/byInbound/:inboundId", a.list)
	g.GET("/tags", a.tags)
	g.GET("/sortMode", a.sortMode)
	g.POST("/sortMode", requirePerm(model.PermPanelSettings), a.sortMode)
	g.POST("/add", requirePerm(model.PermEditInbound), a.save)
	g.POST("/bulk/add", requirePerm(model.PermEditInbound), a.save)
	g.POST("/update/:groupId", requirePerm(model.PermEditInbound), a.save)
	g.POST("/del/:groupId", requirePerm(model.PermEditInbound), a.remove)
	g.POST("/setEnable/:groupId", requirePerm(model.PermEditInbound), a.enable)
	g.POST("/bulk/del", requirePerm(model.PermEditInbound), a.remove)
	g.POST("/bulk/setEnable", requirePerm(model.PermEditInbound), a.enable)
	g.POST("/reorder", requirePerm(model.PermEditInbound), a.reorder)
	return a
}

func (a *HostController) list(c *gin.Context) {
	groups, err := a.hosts.List(session.GetLoginUser(c))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	if raw := c.Param("inboundId"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			jsonMsg(c, I18nWeb(c, "get"), err)
			return
		}
		filtered := []*service.HostGroupView{}
		for _, group := range groups {
			for _, inboundID := range group.InboundIds {
				if inboundID == id {
					filtered = append(filtered, group)
					break
				}
			}
		}
		groups = filtered
	}
	jsonObj(c, groups, nil)
}

func (a *HostController) raw(c *gin.Context) {
	rows, err := a.hosts.Raw(session.GetLoginUser(c))
	jsonObj(c, rows, err)
}

func (a *HostController) get(c *gin.Context) {
	groups, err := a.hosts.List(session.GetLoginUser(c))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	for _, group := range groups {
		if group.GroupId == c.Param("groupId") {
			jsonObj(c, group, nil)
			return
		}
	}
	jsonMsg(c, I18nWeb(c, "get"), errNotOwned)
}

func (a *HostController) tags(c *gin.Context) {
	rows, err := a.hosts.Raw(session.GetLoginUser(c))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "get"), err)
		return
	}
	seen := map[string]bool{}
	tags := []string{}
	for _, h := range rows {
		for _, tag := range h.Tags {
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	sort.Strings(tags)
	jsonObj(c, tags, nil)
}

func (a *HostController) sortMode(c *gin.Context) {
	if c.Request.Method == http.MethodPost {
		var request struct {
			Mode string `json:"mode"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
			return
		}
		jsonMsg(c, I18nWeb(c, "success"), a.hosts.SetSortMode(request.Mode))
		return
	}
	mode, err := a.hosts.SortMode()
	jsonObj(c, mode, err)
}

func (a *HostController) save(c *gin.Context) {
	var request model.HostGroup
	if err := c.ShouldBindJSON(&request); err != nil {
		jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
		return
	}
	rows, err := a.hosts.Save(session.GetLoginUser(c), c.Param("groupId"), &request)
	if err == nil {
		websocket.BroadcastInvalidate(websocket.MessageTypeInbounds)
	}
	jsonObj(c, rows, err)
}

type hostMutation struct {
	Ids      []string `json:"ids"`
	GroupIds []string `json:"groupIds"`
	Enable   bool     `json:"enable"`
}

func (a *HostController) mutate(c *gin.Context, action string) {
	var request hostMutation
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			jsonMsg(c, I18nWeb(c, "somethingWentWrong"), err)
			return
		}
	}
	if id := c.Param("groupId"); id != "" {
		request.Ids = []string{id}
	}
	if len(request.Ids) == 0 {
		request.Ids = request.GroupIds
	}
	err := a.hosts.Mutate(session.GetLoginUser(c), request.Ids, action, request.Enable)
	if err == nil {
		websocket.BroadcastInvalidate(websocket.MessageTypeInbounds)
	}
	jsonMsg(c, I18nWeb(c, "success"), err)
}

func (a *HostController) remove(c *gin.Context)  { a.mutate(c, "delete") }
func (a *HostController) enable(c *gin.Context)  { a.mutate(c, "enable") }
func (a *HostController) reorder(c *gin.Context) { a.mutate(c, "reorder") }
