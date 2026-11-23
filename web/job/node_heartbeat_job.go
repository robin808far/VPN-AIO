package job

import (
	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/web/service"
)

// NodeHeartbeatJob probes every enabled registered node so the master's Nodes
// tab shows online/offline status without a manual refresh.
type NodeHeartbeatJob struct {
	nodeService service.NodeService
}

// Run probes all enabled nodes; a failure on one node must not stop the others.
func (j *NodeHeartbeatJob) Run() {
	defer func() {
		if r := recover(); r != nil {
			logger.Warningf("node heartbeat panic: %v", r)
		}
	}()
	j.nodeService.Heartbeat()
}
