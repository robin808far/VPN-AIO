package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v2/backend"
	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/xray"
)

// SingboxService manages the sing-box core: one process per singbox inbound,
// each with its own config.json under /etc/vpn-aio-singbox/server-<id>.
//
// A singbox inbound stores the STANDARD sing-box inbound object (type, listen,
// listen_port, users, tls, transport, ...) directly as its Settings JSON — the
// panel adds no wrapper, so anything sing-box accepts in its "inbounds" array
// works verbatim. Validation runs `sing-box check` on every render before a
// start; a check failure is logged and the process is not launched.
type SingboxService struct{}

var singboxService = &SingboxService{}

// Serialize config replacement and process reconciliation, including inbound saves.
var singboxRuntimeMu sync.Mutex

func singboxProcName(inboundId int) string {
	return fmt.Sprintf("singbox-server-%d", inboundId)
}

// configDir holds this inbound's config.json.
func (s *SingboxService) configDir(inboundId int) string {
	return filepath.Join(singboxBaseDir(), fmt.Sprintf("server-%d", inboundId))
}

func (s *SingboxService) binaryPath() string {
	return daemonBin("sing-box")
}

// Available reports whether the sing-box binary is bundled for this arch (or
// present on PATH, so a source install still works).
func (s *SingboxService) Available() bool {
	if backend.DaemonPath("sing-box") != "" {
		return true
	}
	_, err := exec.LookPath("sing-box")
	return err == nil
}

// singboxOnlyProtocols are the protocols ONLY sing-box can serve (no Xray
// counterpart exists even in the patched core); anything else is sing-box just
// when the inbound's Core field says so.
var singboxOnlyProtocols = []string{"shadowtls"}

// GetSingboxInbounds returns every inbound the sing-box core serves: the
// sing-box-only protocols plus any Xray-family protocol whose Core is "singbox".
func (s *SingboxService) GetSingboxInbounds() ([]*model.Inbound, error) {
	var inbounds []*model.Inbound
	db := database.GetDB()
	err := db.Model(model.Inbound{}).
		Where("protocol IN ? OR (core = ? AND protocol NOT IN ?)",
			singboxOnlyProtocols, "singbox", singboxOnlyProtocols).
		Find(&inbounds).Error
	return inbounds, err
}

// buildConfig renders the full sing-box config for the process group containing
// inbound. A ShadowTLS front and its panel-managed detour target must be in the
// same config; otherwise sing-box cannot resolve the detour tag.
func (s *SingboxService) buildConfig(inbound *model.Inbound) ([]byte, error) {
	inbounds, err := singboxInboundsWithCandidate(inbound)
	if err != nil {
		return nil, err
	}
	group, err := singboxGroupForCandidate(inbound, inbounds)
	if err != nil {
		return nil, err
	}
	for _, member := range group.Members {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(member.Settings), &object); err != nil || object == nil {
			return nil, fmt.Errorf("inbound %d: settings must be a sing-box JSON object", member.Id)
		}
		var typ string
		if err := json.Unmarshal(object["type"], &typ); err != nil || typ != string(member.Protocol) || typ == "" {
			return nil, fmt.Errorf("inbound %d: sing-box type must match protocol", member.Id)
		}
		if err := validateSingboxLimits(member); err != nil {
			return nil, fmt.Errorf("inbound %d: settings is not a sing-box inbound object: %w", member.Id, err)
		}
	}
	baseText, err := baseConfigService.GetBaseConfig()
	if err != nil {
		return nil, err
	}
	return s.renderConfigWithBase(group.Root, group.Members, baseText)
}

func (s *SingboxService) ensureAPIPort(inbound *model.Inbound) error {
	if inbound.SingboxAPIPort > 0 && inbound.SingboxAPIPort <= 65535 {
		return nil
	}
	port, err := allocateSingboxAPIPort()
	if err != nil {
		return err
	}
	if inbound.Id == 0 {
		inbound.SingboxAPIPort = port
		return nil
	}
	if err := database.GetDB().Model(&model.Inbound{}).Where("id = ?", inbound.Id).
		Update("singbox_api_port", port).Error; err != nil {
		return err
	}
	inbound.SingboxAPIPort = port
	return nil
}

// runtimeInbound projects account enable/quota state into the active users array.
// The stored inbound keeps the user's full credentials for later renewal/reset.
func (s *SingboxService) runtimeInbound(inbound *model.Inbound) (*model.Inbound, error) {
	if inbound == nil {
		return nil, fmt.Errorf("nil sing-box inbound")
	}
	runtime := *inbound
	// cloudflared is a tunnel CLIENT, not a listener with managed accounts --
	// its sing-box settings carry only an admin-set token, so there is no
	// users array to project quota/enable state into. Skip straight to the
	// unmodified settings, same as the save-time exception in
	// validateSingboxLimits (web/service/singbox_inbound.go).
	if inbound.Protocol == "cloudflared" {
		return &runtime, nil
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(inbound.Settings), &root); err != nil || root == nil {
		return nil, fmt.Errorf("inbound %d: invalid sing-box settings", inbound.Id)
	}
	users, ok := root["users"].([]any)
	if !ok {
		return nil, fmt.Errorf("inbound %d: missing sing-box users", inbound.Id)
	}
	names := make([]string, 0, len(users))
	for _, item := range users {
		if user, ok := item.(map[string]any); ok {
			if name := stringValue(user["name"]); name != "" {
				names = append(names, accountKey(name))
			}
		}
	}
	if len(names) == 0 {
		root["users"] = []any{}
		encoded, err := json.MarshalIndent(root, "", "  ")
		runtime.Settings = string(encoded)
		return &runtime, err
	}

	var trafficRows []xray.ClientTraffic
	if err := database.GetDB().Where("LOWER(TRIM(email)) IN ?", names).Find(&trafficRows).Error; err != nil {
		return nil, err
	}
	trafficByEmail := make(map[string]xray.ClientTraffic, len(trafficRows))
	for _, row := range trafficRows {
		trafficByEmail[accountKey(row.Email)] = row
	}
	type membershipState struct {
		Email            string
		AccountEnabled   bool
		MembershipEnable *bool
	}
	var memberships []membershipState
	if err := database.GetDB().Table("account_inbounds").
		Select("accounts.email AS email, accounts.enable AS account_enabled, account_inbounds.enable AS membership_enable").
		Joins("JOIN accounts ON accounts.id = account_inbounds.account_id").
		Where("account_inbounds.inbound_id = ?", inbound.Id).Scan(&memberships).Error; err != nil {
		return nil, err
	}
	membershipByEmail := make(map[string]membershipState, len(memberships))
	for _, row := range memberships {
		membershipByEmail[accountKey(row.Email)] = row
	}
	now := time.Now().UnixMilli()
	filtered := make([]any, 0, len(users))
	for _, item := range users {
		user, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := accountKey(stringValue(user["name"]))
		traffic, hasTraffic := trafficByEmail[key]
		membership, hasMembership := membershipByEmail[key]
		if !hasTraffic || !hasMembership || !membership.AccountEnabled || !traffic.Enable {
			continue
		}
		if membership.MembershipEnable != nil && !*membership.MembershipEnable {
			continue
		}
		if (traffic.Total > 0 && traffic.Up+traffic.Down >= traffic.Total) ||
			(traffic.ExpiryTime > 0 && traffic.ExpiryTime <= now && traffic.Reset == 0) {
			continue
		}
		filtered = append(filtered, item)
	}
	root["users"] = filtered
	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	runtime.Settings = string(encoded)
	return &runtime, nil
}

func (s *SingboxService) renderConfigWithBase(root *model.Inbound, members []*model.Inbound, baseText string) ([]byte, error) {
	if root == nil || root.Tag == "" || root.SingboxAPIPort <= 0 || root.SingboxAPIPort > 65535 {
		return nil, fmt.Errorf("sing-box process group is missing a root tag or stats API port")
	}
	rawInbounds := make([]json.RawMessage, 0, len(members))
	for _, inbound := range members {
		if inbound == nil {
			continue
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(inbound.Settings), &object); err != nil || object == nil {
			return nil, fmt.Errorf("inbound %d: settings must be a sing-box JSON object", inbound.Id)
		}
		if inbound.Tag == "" {
			return nil, fmt.Errorf("inbound %d: missing sing-box inbound tag", inbound.Id)
		}
		tag, err := json.Marshal(inbound.Tag)
		if err != nil {
			return nil, err
		}
		object["tag"] = tag
		if target, err := singboxDetourMember(inbound, members); err != nil {
			return nil, err
		} else if target != nil {
			detour, err := json.Marshal(target.Tag)
			if err != nil {
				return nil, err
			}
			object["detour"] = detour
		}
		rawInbound, err := json.Marshal(object)
		if err != nil {
			return nil, err
		}
		rawInbounds = append(rawInbounds, rawInbound)
	}
	content, err := s.mergeIntoBaseInbounds(rawInbounds, baseText)
	if err != nil {
		return nil, err
	}
	content = []byte(applyCoreConfigOverride("singbox", root.Id, "config.json", string(content)))
	return withSingboxTrafficAPI(content, root, members)
}

func withSingboxTrafficAPI(content []byte, root *model.Inbound, members []*model.Inbound) ([]byte, error) {
	var config map[string]any
	if err := json.Unmarshal(content, &config); err != nil || config == nil {
		return nil, fmt.Errorf("inbound %d: invalid rendered sing-box config", root.Id)
	}
	experimental, _ := config["experimental"].(map[string]any)
	if experimental == nil {
		experimental = map[string]any{}
	}
	// The native listeners and users are owned by the accounts projection. The
	// rendered group already contains runtime-filtered users, so a saved base
	// override cannot restore disabled users or add an unmetered listener.
	rawInbounds := make([]any, 0, len(members))
	statsUsers := make([]string, 0)
	seenUsers := map[string]bool{}
	for _, member := range members {
		var raw map[string]any
		if err := json.Unmarshal([]byte(member.Settings), &raw); err != nil {
			return nil, err
		}
		raw["tag"] = member.Tag
		if target, err := singboxDetourMember(member, members); err != nil {
			return nil, err
		} else if target != nil {
			raw["detour"] = target.Tag
		}
		if users, ok := raw["users"].([]any); ok {
			filtered := make([]any, 0, len(users))
			for _, item := range users {
				user, _ := item.(map[string]any)
				if name := strings.TrimSpace(stringValue(user["name"])); name != "" && !seenUsers[name] {
					seenUsers[name] = true
					statsUsers = append(statsUsers, name)
				}
				filtered = append(filtered, item)
			}
			raw["users"] = filtered
		}
		rawInbounds = append(rawInbounds, raw)
	}
	config["inbounds"] = rawInbounds
	experimental["v2ray_api"] = map[string]any{
		"listen": fmt.Sprintf("127.0.0.1:%d", root.SingboxAPIPort),
		"stats": map[string]any{
			"enabled":  true,
			"inbounds": singboxTags(members),
			"users":    statsUsers,
		},
	}
	config["experimental"] = experimental
	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// checkConfig requires the installed core to validate the exact rendered config.
func (s *SingboxService) checkConfig(path string) error {
	bin := s.binaryPath()
	if !s.Available() {
		return fmt.Errorf("sing-box is not installed; install the core before saving its configuration")
	}
	cmd := exec.Command(bin, "check", "-c", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sing-box check failed: %w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

// generateServerConfig renders and writes the process group's config.json. It
// reports whether the bytes changed, so the caller can restart that process.
func (s *SingboxService) generateServerConfig(inbound *model.Inbound) (bool, error) {
	groups, err := singboxGroupsFromDB()
	if err != nil {
		return false, err
	}
	group := singboxGroupForID(groups, inbound.Id)
	if group == nil {
		return false, fmt.Errorf("inbound %d is not in a sing-box process group", inbound.Id)
	}
	if err := s.ensureAPIPort(group.Root); err != nil {
		return false, err
	}
	dir := s.configDir(group.Root.Id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, err
	}
	runtimeMembers, err := s.runtimeGroupMembers(group)
	if err != nil {
		return false, err
	}
	baseText, err := baseConfigService.GetBaseConfig()
	if err != nil {
		return false, err
	}
	content, err := s.renderConfigWithBase(group.Root, runtimeMembers, baseText)
	if err != nil {
		return false, err
	}
	path := filepath.Join(dir, "config.json")
	old, readErr := os.ReadFile(path)
	if readErr == nil && bytes.Equal(old, content) {
		if err := s.checkConfig(path); err != nil {
			return false, err
		}
		return false, nil
	}
	// Validate a private candidate before replacing the working config or stopping
	// its process. A failed check leaves both untouched.
	candidate, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return false, err
	}
	defer os.Remove(candidate.Name())
	if _, err := candidate.Write(content); err != nil {
		candidate.Close()
		return false, err
	}
	if err := candidate.Close(); err != nil {
		return false, err
	}
	if err := s.checkConfig(candidate.Name()); err != nil {
		return false, err
	}
	if err := os.Rename(candidate.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}

// restartServer (re)starts one inbound's sing-box after validating its config.
func (s *SingboxService) restartServer(inbound *model.Inbound) error {
	dir := s.configDir(inbound.Id)
	name := singboxProcName(inbound.Id)
	if err := s.checkConfig(dir + "/config.json"); err != nil {
		return err
	}
	speedLimitPath := xray.GetSpeedLimitPath()
	if abs, err := filepath.Abs(speedLimitPath); err == nil {
		speedLimitPath = abs
	}
	env := []string{"XRAY_SPEEDLIMIT_FILE=" + speedLimitPath}
	if err := procMgr.Start(name, s.binaryPath(), []string{"run", "-c", dir + "/config.json"}, env, dir); err != nil {
		return fmt.Errorf("sing-box: start %s: %w", name, err)
	}
	// Exec success alone does not mean startup succeeded: bind/TLS errors exit
	// immediately. Cancel the supervisor before restoring a previous config.
	for range 10 {
		time.Sleep(20 * time.Millisecond)
		if !procMgr.IsRunning(name) {
			_ = procMgr.Stop(name)
			return fmt.Errorf("sing-box: %s exited during startup: %s", name, procMgr.Logs(name))
		}
	}
	return nil
}

// InitSingbox brings the sing-box stack up at panel start.
func (s *SingboxService) InitSingbox() {
	if updated, err := s.migrateDetourInboundIDs(); err != nil {
		logger.Warning("sing-box: detour relation backfill failed:", err)
	} else if updated > 0 {
		logger.Infof("sing-box: backfilled %d managed detour relation(s)", updated)
	}
	inbounds, err := s.GetSingboxInbounds()
	if err != nil || len(inbounds) == 0 {
		return
	}
	if err := s.GenerateAllConfigs(); err != nil {
		logger.Warning("sing-box: config generation failed:", err)
		return
	}
	if err := s.RestartServices(); err != nil {
		logger.Warning("sing-box: failed to start services:", err)
	}
}

// RestartServices starts or restarts every enabled singbox inbound, honouring
// maintenance mode: during maintenance this is a no-op so the operator's stop
// holds.
func (s *SingboxService) RestartServices() error {
	singboxRuntimeMu.Lock()
	defer singboxRuntimeMu.Unlock()
	if blocked, _ := singboxMaintenance.Blocked(); blocked {
		logger.Info("sing-box: maintenance active, not starting inbounds")
		return nil
	}
	return s.restartServices()
}

// GenerateAllConfigs renders every enabled singbox inbound's config.json.
func (s *SingboxService) GenerateAllConfigs() error {
	singboxRuntimeMu.Lock()
	defer singboxRuntimeMu.Unlock()
	groups, err := singboxGroupsFromDB()
	if err != nil {
		return err
	}
	var failures []error
	for _, group := range groups {
		if !singboxGroupEnabled(group) {
			continue
		}
		if err := s.applyGroup(group, false); err != nil {
			failures = append(failures, fmt.Errorf("inbound %d: %w", group.Root.Id, err))
		}
	}
	return errors.Join(failures...)
}

// restartServices starts every enabled singbox process group and stops processes
// for groups that vanished or were disabled. sing-box has no hot reload: any
// config change is a restart of that group only. Callers go through the
// exported RestartServices, which honours maintenance mode.
func (s *SingboxService) restartServices() error {
	if updated, err := s.migrateDetourInboundIDs(); err != nil {
		logger.Warning("sing-box: detour relation backfill failed:", err)
	} else if updated > 0 {
		logger.Infof("sing-box: backfilled %d managed detour relation(s)", updated)
	}
	groups, err := singboxGroupsFromDB()
	if err != nil {
		return err
	}
	desired := map[string]bool{}
	var failures []error
	for _, group := range groups {
		if !singboxGroupEnabled(group) {
			continue
		}
		name := singboxProcName(group.Root.Id)
		desired[name] = true
		if err := s.applyGroup(group, true); err != nil {
			failures = append(failures, fmt.Errorf("inbound %d: %w", group.Root.Id, err))
		}
	}
	for _, name := range procMgr.namesWithPrefix("singbox-server-") {
		if !desired[name] {
			_ = procMgr.Stop(name)
		}
	}
	return errors.Join(failures...)
}

// EnsureServicesRunning starts any enabled singbox inbound that has no running
// process, leaving running instances untouched. Maintenance mode holds them
// down: the operator's deliberate stop is not undone by panel starts or
// inbound saves.
func (s *SingboxService) EnsureServicesRunning() error {
	singboxRuntimeMu.Lock()
	defer singboxRuntimeMu.Unlock()
	if blocked, _ := singboxMaintenance.Blocked(); blocked {
		return nil
	}
	groups, err := singboxGroupsFromDB()
	if err != nil {
		return err
	}
	var failures []error
	for _, group := range groups {
		if !singboxGroupEnabled(group) {
			continue
		}
		name := singboxProcName(group.Root.Id)
		if procMgr.IsRunning(name) {
			continue
		}
		if err := s.applyGroup(group, false); err != nil {
			failures = append(failures, fmt.Errorf("inbound %d: %w", group.Root.Id, err))
		}
	}
	return errors.Join(failures...)
}

// RefreshInbound applies account and quota state to one committed inbound.
func (s *SingboxService) RefreshInbound(id int) error {
	singboxRuntimeMu.Lock()
	defer singboxRuntimeMu.Unlock()
	if blocked, _ := singboxMaintenance.Blocked(); blocked {
		return nil
	}
	inbound, err := (&InboundService{}).GetInbound(id)
	if err != nil {
		return err
	}
	if err := s.disableExpiredInbound(inbound); err != nil {
		return err
	}
	groups, err := singboxGroupsFromDB()
	if err != nil {
		return err
	}
	group := singboxGroupForID(groups, id)
	if group == nil {
		return fmt.Errorf("inbound %d is not in a sing-box process group", id)
	}
	return s.applyGroup(group, false)
}

// RefreshServices reconciles DB quota state with each sing-box process. It runs
// after traffic commits and on idle ticks so expired listeners cannot stay live.
func (s *SingboxService) RefreshServices() error {
	singboxRuntimeMu.Lock()
	defer singboxRuntimeMu.Unlock()
	if blocked, _ := singboxMaintenance.Blocked(); blocked {
		return nil
	}
	groups, err := singboxGroupsFromDB()
	if err != nil {
		return err
	}
	desired := make(map[string]bool, len(groups))
	var failures []error
	for _, group := range groups {
		for _, member := range group.Members {
			if err := s.disableExpiredInbound(member); err != nil {
				failures = append(failures, fmt.Errorf("inbound %d quota state: %w", member.Id, err))
			}
		}
		if !singboxGroupEnabled(group) {
			continue
		}
		name := singboxProcName(group.Root.Id)
		desired[name] = true
		if err := s.applyGroup(group, false); err != nil {
			failures = append(failures, fmt.Errorf("inbound %d: %w", group.Root.Id, err))
		}
	}
	for _, name := range procMgr.namesWithPrefix("singbox-server-") {
		if !desired[name] {
			if err := procMgr.Stop(name); err != nil {
				failures = append(failures, fmt.Errorf("stop %s: %w", name, err))
			}
		}
	}
	return errors.Join(failures...)
}

func (s *SingboxService) disableExpiredInbound(inbound *model.Inbound) error {
	if inbound == nil || !inbound.Enable {
		return nil
	}
	now := time.Now().UnixMilli()
	depleted := (inbound.Total > 0 && inbound.Up+inbound.Down >= inbound.Total) ||
		(inbound.ExpiryTime > 0 && inbound.ExpiryTime <= now)
	if !depleted {
		return nil
	}
	if inbound.Id == 0 {
		inbound.Enable = false
		return nil
	}
	if err := database.GetDB().Model(&model.Inbound{}).Where("id = ? AND enable = ?", inbound.Id, true).
		Update("enable", false).Error; err != nil {
		return err
	}
	inbound.Enable = false
	return nil
}

// GetTraffic reads per-inbound V2Ray-compatible stats from running sing-box
// instances. Every user record is stamped with its source inbound before billing.
func (s *SingboxService) GetTraffic() ([]*xray.Traffic, []*xray.ClientTraffic, error) {
	groups, err := singboxGroupsFromDB()
	if err != nil {
		return nil, nil, err
	}
	var inboundTraffic []*xray.Traffic
	var clientTraffic []*xray.ClientTraffic
	var failures []error
	var trafficRows []xray.ClientTraffic
	if err := database.GetDB().Select("email").Find(&trafficRows).Error; err != nil {
		return nil, nil, err
	}
	canonical := make(map[string]string, len(trafficRows))
	for _, row := range trafficRows {
		canonical[accountKey(row.Email)] = row.Email
	}
	for _, group := range groups {
		if !singboxGroupEnabled(group) {
			continue
		}
		// Keep lifecycle sweeps running for idle or fully depleted inbounds. A
		// chained group has one process but each panel inbound still needs its
		// own traffic row in the normal accounting path.
		for _, member := range group.Members {
			inboundTraffic = append(inboundTraffic, &xray.Traffic{IsInbound: true, Tag: member.Tag})
		}
		if !procMgr.IsRunning(singboxProcName(group.Root.Id)) {
			continue
		}
		if group.Root.SingboxAPIPort <= 0 {
			failures = append(failures, fmt.Errorf("inbound %d has no stats API port", group.Root.Id))
			continue
		}
		var api xray.XrayAPI
		if err := api.InitStats(group.Root.SingboxAPIPort); err != nil {
			failures = append(failures, fmt.Errorf("inbound %d stats API: %w", group.Root.Id, err))
			continue
		}
		inboundRows, users, queryErr := api.GetTraffic(true)
		api.Close()
		if queryErr != nil {
			failures = append(failures, fmt.Errorf("inbound %d stats query: %w", group.Root.Id, queryErr))
			continue
		}
		inboundTraffic = append(inboundTraffic, inboundRows...)
		for _, user := range users {
			// sing-box's V2Ray-compatible user stat has no inbound component.
			// A chained process may carry the same account on both sides of the
			// detour, so guessing a member would misfile the multiplier/breakdown.
			// InboundId=0 bills the account once and deliberately leaves the
			// per-inbound breakdown unattributed.
			user.InboundId = 0
			user.CoreCounted = false
			if email := canonical[accountKey(user.Email)]; email != "" {
				user.Email = email
			}
		}
		clientTraffic = append(clientTraffic, users...)
	}
	return inboundTraffic, clientTraffic, errors.Join(failures...)
}

// reconcileInbound touches the committed process group supplied by its save
// path. Caller holds singboxRuntimeMu.
func (s *SingboxService) reconcileInbound(inbound *model.Inbound) error {
	groups, err := singboxGroupsFromDB()
	if err != nil {
		return err
	}
	group := singboxGroupForID(groups, inbound.Id)
	if group == nil {
		return fmt.Errorf("inbound %d is not in a sing-box process group", inbound.Id)
	}
	return s.applyGroup(group, false)
}

func (s *SingboxService) applyGroup(group *singboxProcessGroup, force bool) error {
	if group == nil || group.Root == nil {
		return fmt.Errorf("nil sing-box process group")
	}
	for _, member := range group.Members {
		if err := s.disableExpiredInbound(member); err != nil {
			return err
		}
	}
	if !singboxGroupEnabled(group) {
		return s.stopGroup(group)
	}
	runtimeMembers, err := s.runtimeGroupMembers(group)
	if err != nil {
		return err
	}
	entries := 0
	emptyMember := 0
	for _, runtime := range runtimeMembers {
		// cloudflared carries no managed users at all -- it is not an empty
		// managed-user member in the Shadowsocks/ShadowTLS sense this guard
		// exists for, so it must not count against entries/emptyMember below.
		if runtime.Protocol == "cloudflared" {
			continue
		}
		items, _ := parseSettingsClients(runtime.Settings)
		entries += len(items)
		if len(items) == 0 {
			emptyMember++
		}
	}
	allCloudflared := true
	for _, runtime := range runtimeMembers {
		if runtime.Protocol != "cloudflared" {
			allCloudflared = false
			break
		}
	}
	if !allCloudflared && (entries == 0 || emptyMember > 0) {
		// An empty Shadowsocks/ShadowTLS user list can enable shared-password
		// authentication. A chained process must also stop when any member has
		// no enabled managed users, or an empty ShadowTLS front could remain
		// reachable while its target still has active users.
		return s.stopGroup(group)
	}
	root := group.Root
	path := filepath.Join(s.configDir(root.Id), "config.json")
	previous, readErr := os.ReadFile(path)
	wasRunning := procMgr.IsRunning(singboxProcName(root.Id))
	changed, err := s.generateServerConfig(root)
	if err != nil {
		return err
	}
	if blocked, _ := singboxMaintenance.Blocked(); blocked {
		return nil
	}
	if !force && !changed && wasRunning {
		return nil
	}
	// A member may previously have owned a standalone process before it became
	// part of this chain. Stop those old names before starting the group owner.
	for _, member := range group.Members {
		if member.Id != root.Id {
			_ = procMgr.Stop(singboxProcName(member.Id))
		}
	}
	if err := s.restartServer(root); err != nil {
		_ = procMgr.Stop(singboxProcName(root.Id))
		if readErr == nil {
			if restoreErr := os.WriteFile(path, previous, 0o600); restoreErr != nil {
				return errors.Join(err, restoreErr)
			}
			if wasRunning {
				return errors.Join(err, s.restartServer(root))
			}
		} else {
			_ = os.Remove(path)
		}
		return err
	}
	return nil
}

func (s *SingboxService) stopGroup(group *singboxProcessGroup) error {
	if group == nil {
		return nil
	}
	var failures []error
	for _, member := range group.Members {
		if member == nil {
			continue
		}
		if err := procMgr.Stop(singboxProcName(member.Id)); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// StopServices stops all sing-box child processes.
func (s *SingboxService) StopServices() error {
	procMgr.StopByPrefix("singbox-server-")
	return nil
}

// SetupRouting is a no-op: sing-box is a userspace relay with no tunnel, so
// there are no nftables rules or kernel modules to install. Kept so the service
// matches the shape of its siblings.
func (s *SingboxService) SetupRouting() error { return nil }
