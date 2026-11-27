package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"gorm.io/gorm"
)

// singboxProcessGroup is the set of panel inbounds that must live in one
// sing-box process because at least one inbound detours to another member.
// The root is only the stable process/config owner; it is not necessarily the
// first protocol a client enters.
type singboxProcessGroup struct {
	Root    *model.Inbound
	Members []*model.Inbound
}

func singboxGroupsFromDB() ([]*singboxProcessGroup, error) {
	var inbounds []*model.Inbound
	if err := database.GetDB().Model(model.Inbound{}).
		Where("protocol IN ? OR (core = ? AND protocol NOT IN ?)",
			singboxOnlyProtocols, "singbox", singboxOnlyProtocols).
		Find(&inbounds).Error; err != nil {
		return nil, err
	}
	return singboxGroupsFromInbounds(inbounds)
}

// singboxGroupsFromInbounds resolves panel-managed detours and returns
// connected components. Keeping this pure makes relation validation testable
// without a running sing-box binary or a database.
func singboxGroupsFromInbounds(inbounds []*model.Inbound) ([]*singboxProcessGroup, error) {
	byID := make(map[int]*model.Inbound, len(inbounds))
	byTag := make(map[string]*model.Inbound, len(inbounds))
	for _, inbound := range inbounds {
		if inbound == nil {
			continue
		}
		byID[inbound.Id] = inbound
		if tag := strings.TrimSpace(inbound.Tag); tag != "" {
			byTag[tag] = inbound
		}
	}

	next := make(map[int]int)
	for _, inbound := range inbounds {
		if inbound == nil {
			continue
		}
		targetID, targetTag, err := singboxDetourTarget(inbound)
		if err != nil {
			return nil, err
		}
		if targetID == 0 && targetTag == "" {
			continue
		}
		var target *model.Inbound
		if targetID != 0 {
			target = byID[targetID]
			if target == nil {
				return nil, fmt.Errorf("sing-box inbound %d detour target %d was not found", inbound.Id, targetID)
			}
		} else {
			target = byTag[targetTag]
			if target == nil && inbound.SingboxDetourInboundID > 0 {
				// A target's generated tag can change when its listener is
				// edited. The persisted id lets the existing chain survive that
				// rename; an explicitly changed tag still wins when it resolves.
				target = byID[inbound.SingboxDetourInboundID]
			}
			if target == nil {
				return nil, fmt.Errorf("sing-box inbound %d detour target %q was not found", inbound.Id, targetTag)
			}
		}
		if target.Id == inbound.Id {
			return nil, fmt.Errorf("sing-box inbound %d cannot detour to itself", inbound.Id)
		}
		if !isSingboxInbound(target) {
			return nil, fmt.Errorf("sing-box inbound %d detour target %d is not served by sing-box", inbound.Id, target.Id)
		}
		inbound.SingboxDetourInboundID = target.Id
		next[inbound.Id] = target.Id
	}

	// A cycle cannot be represented by sing-box detour routing and otherwise
	// would make component ownership depend on map iteration order.
	state := make(map[int]uint8, len(next))
	var visit func(int) error
	visit = func(id int) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("sing-box detour chain contains a cycle at inbound %d", id)
		case 2:
			return nil
		}
		state[id] = 1
		if target := next[id]; target != 0 {
			if err := visit(target); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range next {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	adj := make(map[int][]int, len(inbounds))
	for _, inbound := range inbounds {
		if inbound == nil {
			continue
		}
		adj[inbound.Id] = nil
	}
	for from, to := range next {
		adj[from] = append(adj[from], to)
		adj[to] = append(adj[to], from)
	}
	ids := make([]int, 0, len(adj))
	for id := range adj {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	seen := make(map[int]bool, len(ids))
	groups := make([]*singboxProcessGroup, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		queue := []int{id}
		seen[id] = true
		component := make([]int, 0, 1)
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			component = append(component, current)
			for _, neighbour := range adj[current] {
				if !seen[neighbour] {
					seen[neighbour] = true
					queue = append(queue, neighbour)
				}
			}
		}
		sort.Ints(component)
		members := make([]*model.Inbound, 0, len(component))
		for _, memberID := range component {
			members = append(members, byID[memberID])
		}
		groups = append(groups, &singboxProcessGroup{Root: members[0], Members: members})
	}
	return groups, nil
}

func singboxDetourTarget(inbound *model.Inbound) (int, string, error) {
	if inbound == nil {
		return 0, "", fmt.Errorf("nil sing-box inbound in detour chain")
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(inbound.Settings), &root); err != nil || root == nil {
		return 0, "", fmt.Errorf("inbound %d: invalid sing-box settings while reading detour", inbound.Id)
	}
	var tag string
	if raw, ok := root["detour"]; ok {
		if err := json.Unmarshal(raw, &tag); err != nil {
			return 0, "", fmt.Errorf("inbound %d: sing-box detour must be a string", inbound.Id)
		}
		tag = strings.TrimSpace(tag)
	}
	// The raw sing-box object is the editable source of truth. The persisted id
	// is a fallback for a chain created by the panel when the target's generated
	// tag changes; an explicit raw detour must still be able to retarget it.
	if tag == "" && inbound.SingboxDetourInboundID > 0 {
		return inbound.SingboxDetourInboundID, "", nil
	}
	return 0, tag, nil
}

func singboxDetourMember(inbound *model.Inbound, members []*model.Inbound) (*model.Inbound, error) {
	if inbound == nil {
		return nil, fmt.Errorf("nil sing-box inbound in detour chain")
	}
	targetID, targetTag, err := singboxDetourTarget(inbound)
	if err != nil {
		return nil, err
	}
	if targetID == 0 && targetTag == "" {
		return nil, nil
	}
	for _, member := range members {
		if member == nil {
			continue
		}
		if (targetID != 0 && member.Id == targetID) || (targetID == 0 && member.Tag == targetTag) {
			return member, nil
		}
	}
	// The raw tag may be stale after the target's listener was edited. The
	// panel-created stable id remains authoritative within this process group.
	if inbound.SingboxDetourInboundID > 0 {
		for _, member := range members {
			if member != nil && member.Id == inbound.SingboxDetourInboundID {
				return member, nil
			}
		}
	}
	return nil, fmt.Errorf("inbound %d: detour target is outside its process group", inbound.Id)
}

func singboxGroupForID(groups []*singboxProcessGroup, id int) *singboxProcessGroup {
	for _, group := range groups {
		for _, member := range group.Members {
			if member != nil && member.Id == id {
				return group
			}
		}
	}
	return nil
}

func singboxGroupForCandidate(candidate *model.Inbound, inbounds []*model.Inbound) (*singboxProcessGroup, error) {
	if candidate == nil {
		return nil, fmt.Errorf("nil sing-box inbound")
	}
	groups, err := singboxGroupsFromInbounds(inbounds)
	if err != nil {
		return nil, err
	}
	group := singboxGroupForID(groups, candidate.Id)
	if group == nil {
		return nil, fmt.Errorf("sing-box inbound %d is not in a process group", candidate.Id)
	}
	return group, nil
}

func singboxInboundsWithCandidate(candidate *model.Inbound) ([]*model.Inbound, error) {
	var inbounds []*model.Inbound
	if err := database.GetDB().Model(model.Inbound{}).
		Where("protocol IN ? OR (core = ? AND protocol NOT IN ?)",
			singboxOnlyProtocols, "singbox", singboxOnlyProtocols).
		Find(&inbounds).Error; err != nil {
		return nil, err
	}
	for i, inbound := range inbounds {
		if inbound != nil && candidate.Id != 0 && inbound.Id == candidate.Id {
			inbounds[i] = candidate
			return inbounds, nil
		}
	}
	inbounds = append(inbounds, candidate)
	return inbounds, nil
}

func singboxGroupIDs(group *singboxProcessGroup) []int {
	if group == nil {
		return nil
	}
	ids := make([]int, 0, len(group.Members))
	for _, member := range group.Members {
		if member != nil {
			ids = append(ids, member.Id)
		}
	}
	sort.Ints(ids)
	return ids
}

func sameSingboxGroup(a, b *singboxProcessGroup) bool {
	left, right := singboxGroupIDs(a), singboxGroupIDs(b)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func singboxGroupEnabled(group *singboxProcessGroup) bool {
	if group == nil || group.Root == nil || len(group.Members) == 0 {
		return false
	}
	for _, member := range group.Members {
		if member == nil || !member.Enable {
			return false
		}
	}
	return true
}

func singboxTags(members []*model.Inbound) []string {
	tags := make([]string, 0, len(members))
	for _, member := range members {
		if member != nil && strings.TrimSpace(member.Tag) != "" {
			tags = append(tags, member.Tag)
		}
	}
	return tags
}

func (s *SingboxService) runtimeGroupMembers(group *singboxProcessGroup) ([]*model.Inbound, error) {
	if group == nil {
		return nil, fmt.Errorf("nil sing-box process group")
	}
	runtimeMembers := make([]*model.Inbound, 0, len(group.Members))
	for _, member := range group.Members {
		runtime, err := s.runtimeInbound(member)
		if err != nil {
			return nil, err
		}
		runtimeMembers = append(runtimeMembers, runtime)
	}
	return runtimeMembers, nil
}

// migrateDetourInboundIDs backfills the stable target relation for existing
// panel-managed detours. Resolution is performed before the transaction, so an
// invalid graph leaves every row untouched; only the internal relation column
// is updated and the caller-owned sing-box settings remain unchanged.
func (s *SingboxService) migrateDetourInboundIDs() (int, error) {
	db := database.GetDB()
	if db == nil {
		return 0, fmt.Errorf("database is not initialized")
	}
	var inbounds []*model.Inbound
	if err := db.Model(&model.Inbound{}).
		Where("protocol IN ? OR (core = ? AND protocol NOT IN ?)",
			singboxOnlyProtocols, "singbox", singboxOnlyProtocols).
		Find(&inbounds).Error; err != nil {
		return 0, err
	}
	previous := make(map[int]int, len(inbounds))
	for _, inbound := range inbounds {
		if inbound != nil {
			previous[inbound.Id] = inbound.SingboxDetourInboundID
		}
	}
	if _, err := singboxGroupsFromInbounds(inbounds); err != nil {
		return 0, err
	}
	var changed []*model.Inbound
	for _, inbound := range inbounds {
		if inbound != nil && inbound.SingboxDetourInboundID != previous[inbound.Id] {
			changed = append(changed, inbound)
		}
	}
	if len(changed) == 0 {
		return 0, nil
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, inbound := range changed {
			if err := tx.Model(&model.Inbound{}).Where("id = ?", inbound.Id).
				Update("singbox_detour_inbound_id", inbound.SingboxDetourInboundID).Error; err != nil {
				return fmt.Errorf("persist detour target for inbound %d: %w", inbound.Id, err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(changed), nil
}

func clearSingboxDetourReferences(tx *gorm.DB, targetID int, targetTag string) error {
	var inbounds []model.Inbound
	if err := tx.Model(&model.Inbound{}).
		Where("singbox_detour_inbound_id = ? OR protocol = ? OR core = ?", targetID, "shadowtls", "singbox").
		Find(&inbounds).Error; err != nil {
		return err
	}
	for i := range inbounds {
		inbound := &inbounds[i]
		if inbound.Id == targetID || !isSingboxInbound(inbound) {
			continue
		}
		changed := inbound.SingboxDetourInboundID == targetID
		var root map[string]any
		if err := json.Unmarshal([]byte(inbound.Settings), &root); err == nil && root != nil {
			if tag, _ := root["detour"].(string); targetTag != "" && strings.TrimSpace(tag) == targetTag {
				delete(root, "detour")
				encoded, marshalErr := json.Marshal(root)
				if marshalErr != nil {
					return marshalErr
				}
				inbound.Settings = string(encoded)
				changed = true
			}
		}
		if !changed {
			continue
		}
		inbound.SingboxDetourInboundID = 0
		if err := tx.Model(&model.Inbound{}).Where("id = ?", inbound.Id).
			Updates(map[string]any{
				"settings":                  inbound.Settings,
				"singbox_detour_inbound_id": 0,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}
