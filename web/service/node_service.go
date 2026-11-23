package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v2/config"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"

	"gorm.io/gorm"
)

var ErrNodeNotFound = errors.New("node not found")

// NodeStatus is what a remote panel answers on its /nodeapi/status endpoint.
type NodeStatus struct {
	Version   string `json:"version"`
	XrayRunning bool `json:"xrayRunning"`
	Inbounds  int64  `json:"inbounds"`
	Clients   int64  `json:"clients"`
	Traffic   int64  `json:"trafficTotal"`
}

// NodeService manages the remote panels registered on this master: CRUD,
// reachability probes and the heartbeat that keeps Status/LastSeen current.
type NodeService struct{}

// Get returns one node by id.
func (s *NodeService) Get(id int) (*model.Node, error) {
	node := &model.Node{}
	if err := database.GetDB().First(node, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return node, nil
}

// List returns every node ordered by id.
func (s *NodeService) List() ([]*model.Node, error) {
	nodes := make([]*model.Node, 0)
	err := database.GetDB().Order("id asc").Find(&nodes).Error
	return nodes, err
}

// Add registers a remote panel and immediately probes it so the caller sees a
// reachable/unreachable verdict right away instead of waiting for the heartbeat.
func (s *NodeService) Add(node *model.Node) (*model.Node, error) {
	s.normalize(node)
	if node.Address == "" {
		return nil, errors.New("node address is required")
	}
	if err := database.GetDB().Create(node).Error; err != nil {
		return nil, err
	}
	s.Probe(node)
	return node, nil
}

// Update applies the editable fields of an existing node. An empty token keeps
// the stored one, so editing other fields never forces re-entering the secret.
func (s *NodeService) Update(id int, form *model.Node) (*model.Node, error) {
	node, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	s.normalize(form)
	node.Scheme = form.Scheme
	node.Address = form.Address
	node.Port = form.Port
	node.BasePath = form.BasePath
	node.SyncMode = form.SyncMode
	node.Enabled = form.Enabled
	if strings.TrimSpace(form.Token) != "" {
		node.Token = form.Token
	}
	if err := database.GetDB().Save(node).Error; err != nil {
		return nil, err
	}
	s.Probe(node)
	return node, nil
}

// Del removes a node. Remote inbounds are untouched, matching the reference:
// deleting the registration never destroys data on the child panel.
func (s *NodeService) Del(id int) error {
	result := database.GetDB().Delete(&model.Node{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNodeNotFound
	}
	return nil
}

// Probe calls the remote panel's /nodeapi/status and updates the stored status,
// version, inbound count and last-seen time.
func (s *NodeService) Probe(node *model.Node) *NodeStatus {
	status, err := s.Status(node)
	now := time.Now().Unix()
	node.LastSeen = now
	if err != nil {
		node.Status = "down"
		logger.Warningf("node %s probe failed: %v", node.Name, err)
	} else {
		node.Status = "online"
		node.Version = status.Version
		node.Inbounds = int(status.Inbounds)
	}
	if err := database.GetDB().Save(node).Error; err != nil {
		logger.Warningf("node %s probe persist failed: %v", node.Name, err)
	}
	return status
}

// Heartbeat probes every enabled node in sequence.
func (s *NodeService) Heartbeat() {
	nodes, err := s.List()
	if err != nil {
		return
	}
	for _, node := range nodes {
		if !node.Enabled {
			continue
		}
		s.Probe(node)
	}
}

// Status fetches the remote /nodeapi/status payload.
func (s *NodeService) Status(node *model.Node) (*NodeStatus, error) {
	body, err := s.get(node, "status")
	if err != nil {
		return nil, err
	}
	status := &NodeStatus{}
	if err := json.Unmarshal(body, status); err != nil {
		return nil, err
	}
	return status, nil
}

// RemoteInbounds lists the remote panel's inbounds (summary fields only).
func (s *NodeService) RemoteInbounds(node *model.Node) ([]map[string]any, error) {
	body, err := s.get(node, "inbounds")
	if err != nil {
		return nil, err
	}
	inbounds := make([]map[string]any, 0)
	if err := json.Unmarshal(body, &inbounds); err != nil {
		return nil, err
	}
	return inbounds, nil
}

// get performs an authenticated GET against the node API and returns the body.
func (s *NodeService) get(node *model.Node, path string) ([]byte, error) {
	if strings.TrimSpace(node.Token) == "" {
		return nil, errors.New("node has no API token")
	}
	req, err := http.NewRequest(http.MethodGet, s.url(node, path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+node.Token)
	req.Header.Set("User-Agent", "VPN-AIO-Master/"+config.GetVersion())
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("node %s/nodeapi/%s returned %d", s.url(node, ""), path, resp.StatusCode)
	}
	return body, nil
}

// url builds the absolute nodeapi URL for a node.
func (s *NodeService) url(node *model.Node, path string) string {
	scheme := node.Scheme
	if scheme == "" {
		scheme = "http"
	}
	host := node.Address
	if node.Port > 0 {
		host = host + ":" + strconv.Itoa(node.Port)
	}
	base := strings.TrimSuffix(node.BasePath, "/")
	return (&url.URL{Scheme: scheme, Host: host, Path: base + "/nodeapi/" + path}).String()
}

// normalize defaults scheme, base path and sync mode so callers may omit them.
func (s *NodeService) normalize(node *model.Node) {
	node.Scheme = strings.TrimSpace(node.Scheme)
	if node.Scheme != "https" {
		node.Scheme = "http"
	}
	if node.BasePath == "" {
		node.BasePath = "/"
	}
	if !strings.HasPrefix(node.BasePath, "/") {
		node.BasePath = "/" + node.BasePath
	}
	if node.SyncMode != "selected" {
		node.SyncMode = "all"
	}
	node.Address = strings.TrimSpace(node.Address)
	node.Name = strings.TrimSpace(node.Name)
}
