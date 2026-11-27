package controller

import (
	"net/http"

	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/gin-gonic/gin"
)

type NodeController struct {
	service *service.NodeService
}

func NewNodeController(g *gin.RouterGroup, service *service.NodeService) {
	controller := &NodeController{service: service}
	group := g.Group("/nodes")
	group.Use(requirePerm(model.PermAccessNodes))
	group.GET("", controller.list)
	group.POST("", controller.add)
	// Add more routes as needed
}

func (c *NodeController) list(ctx *gin.Context) {
	nodes, err := c.service.GetAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, nodes)
}

// Implement add, update, delete methods similarly
