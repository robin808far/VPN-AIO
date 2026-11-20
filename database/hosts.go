package database

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func migrateHosts() error {
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.HistoryOfSeeders{}).Where("seeder_name = ?", "HostsFromExternalProxyV3").Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return nil
		}
		var inbounds []model.Inbound
		if err := tx.Find(&inbounds).Error; err != nil {
			return err
		}
		for i := range inbounds {
			if err := ImportLegacyHosts(tx, &inbounds[i]); err != nil {
				return err
			}
		}
		return tx.Create(&model.HistoryOfSeeders{SeederName: "HostsFromExternalProxyV3"}).Error
	})
}

func legacyHostRows(inbound *model.Inbound) map[string]*model.Host {
	rows := map[string]*model.Host{}
	// Non-Xray protocols keep their endpoint lists inside protocol settings.
	sources := map[string]string{"stream": inbound.StreamSettings}
	switch inbound.Protocol {
	case model.WGC, model.AWG, model.SSH, model.MTPROTO, model.OPENVPN, model.L2TP, model.PPTP, model.OPENCONNECT, model.SSTP, model.IKEV2:
		sources = map[string]string{"settings": inbound.Settings}
	}
	for source, raw := range sources {
		var settings struct {
			ExternalProxy []struct {
				Dest          string   `json:"dest"`
				Port          int      `json:"port"`
				ForceTls      string   `json:"forceTls"`
				Remark        string   `json:"remark"`
				Sni           string   `json:"sni"`
				Fingerprint   string   `json:"fingerprint"`
				Alpn          []string `json:"alpn"`
				EchConfigList string   `json:"echConfigList"`
			} `json:"externalProxy"`
		}
		if json.Unmarshal([]byte(raw), &settings) != nil {
			continue
		}
		for i, ep := range settings.ExternalProxy {
			security := ep.ForceTls
			if security != "tls" && security != "none" {
				security = "same"
			}
			key := fmt.Sprintf("%s:%d", source, i)
			h := &model.Host{InboundId: inbound.Id, GroupId: uuid.NewString(), SortOrder: i,
				Address: ep.Dest, Port: ep.Port, Security: security, Remark: ep.Remark,
				Sni: ep.Sni, Fingerprint: ep.Fingerprint, Alpn: ep.Alpn, EchConfigList: ep.EchConfigList,
				RandomCount: 1, LegacySource: key}
			rows[key] = h
		}
	}
	return rows
}

// ImportLegacyHosts is also used for imports created after the startup migration ran.
func ImportLegacyHosts(tx *gorm.DB, inbound *model.Inbound) error {
	rows := legacyHostRows(inbound)
	for _, h := range rows {
		var count int64
		if err := tx.Model(&model.Host{}).Where("inbound_id = ? AND legacy_source = ?", inbound.Id, h.LegacySource).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Create(h).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func legacyHostEqual(a, b *model.Host) bool {
	x, y := *a, *b
	x.GroupId, y.GroupId = "", ""
	return reflect.DeepEqual(x, y)
}

// SyncLegacyHosts updates untouched migrated rows, preserving manual edits and deleted rows.
func SyncLegacyHosts(tx *gorm.DB, old, next *model.Inbound) error {
	before, after := legacyHostRows(old), legacyHostRows(next)
	for key, desired := range after {
		prior, existed := before[key]
		if existed && legacyHostEqual(prior, desired) {
			continue
		}
		var existing model.Host
		err := tx.Where("inbound_id = ? AND legacy_source = ?", next.Id, key).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if existed {
				continue
			}
			if err := tx.Create(desired).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			desired.Id, desired.GroupId, desired.CreatedAt = existing.Id, existing.GroupId, existing.CreatedAt
			if err := tx.Save(desired).Error; err != nil {
				return err
			}
		}
	}
	for key := range before {
		if _, exists := after[key]; !exists {
			if err := DeleteHosts(tx, "inbound_id = ? AND legacy_source = ?", next.Id, key); err != nil {
				return err
			}
		}
	}
	return nil
}

func DeleteHosts(tx *gorm.DB, query string, args ...any) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("host deletion requires a scope")
	}
	var rows []model.Host
	if err := tx.Where(query, args...).Find(&rows).Error; err != nil {
		return err
	}
	for _, h := range rows {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&model.HostTombstone{HostId: h.Id, InboundId: h.InboundId}).Error; err != nil {
			return err
		}
	}
	return tx.Where(query, args...).Delete(&model.Host{}).Error
}
