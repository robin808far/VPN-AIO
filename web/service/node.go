package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/util/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NodeService struct {
	db *gorm.DB
}

func NewNodeService(db *gorm.DB) *NodeService {
	return &NodeService{db: db}
}

func (s *NodeService) Add(node *model.Node) error {
	return s.db.Create(node).Error
}

func (s *NodeService) GetAll() ([]*model.Node, error) {
	var nodes []*model.Node
	if err := s.db.Order("id ASC").Find(&nodes).Error;
	return nodes, err
}

// Additional methods (Update, Delete, etc.) will be added here
