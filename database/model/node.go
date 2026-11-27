package model

type Node struct {
    ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
    Name                string    `gorm:"uniqueIndex" json:"name" validate:"required"`
    Remark              *string   `gorm:"type:text" json:"remark"`
    Scheme              string    `gorm:"default:https" json:"scheme" validate:"omitempty,oneof=http https"`
    Address             string    `gorm:"not null" json:"address" validate:"required"`
    Port                int       `gorm:"default:2053" json:"port" validate:"gte=1,lte=65535"`
    BasePath            *string   `gorm:"type:text" json:"basePath"`
    ApiToken            string    `gorm:"column:api_token;not null" json:"-" validate:"required"`
    Enable              bool      `gorm:"default:true" json:"enable"`
    AllowPrivateAddress bool      `gorm:"default:false" json:"allowPrivateAddress"`
    TlsVerifyMode       string    `gorm:"column:tls_verify_mode;default:verify" json:"tlsVerifyMode" validate:"omitempty,oneof=verify skip pin mtls"`
    PinnedCertSha256    *string   `gorm:"column:pinned_cert_sha256" json:"pinnedCertSha256"`
    InboundSyncMode     string    `gorm:"column:inbound_sync_mode;default:all" json:"inboundSyncMode" validate:"omitempty,oneof=all selected"`
    InboundTags         []string  `gorm:"type:text;serializer:json" json:"inboundTags"`
    OutboundTag         *string   `gorm:"column:outbound_tag" json:"outboundTag"`
    Guid               string    `gorm:"uniqueIndex" json:"guid"`
    Status              string    `gorm:"default:unknown" json:"status"`
    LastHeartbeat       int64     `gorm:"default:0" json:"lastHeartbeat"`
    LatencyMs           *int      `gorm:"type:integer" json:"latencyMs"`
    XrayVersion         *string   `gorm:"type:text" json:"xrayVersion"`
    PanelVersion        *string   `gorm:"column:panel_version" json:"panelVersion"`
    CpuPct              *float64  `gorm:"type:real" json:"cpuPct"`
    MemPct              *float64  `gorm:"type:real" json:"memPct"`
    UptimeSecs          *uint64   `gorm:"type:integer" json:"uptimeSecs"`
    UnavailableMetrics  []string  `gorm:"type:text;serializer:json" json:"unavailableMetrics,omitempty"`
    NetUp               uint64    `gorm:"column:net_up;default:0" json:"netUp"`
    NetDown             uint64    `gorm:"column:net_down;default:0" json:"netDown"`
    LastError           *string   `gorm:"type:text" json:"lastError"`
    XrayState           *string   `gorm:"column:xray_state" json:"xrayState"`
    XrayError           *string   `gorm:"column:xray_error" json:"xrayError"`
    ConfigDirty         bool      `gorm:"default:false" json:"configDirty"`
    ConfigDirtyAt       *int64    `gorm:"type:integer" json:"configDirtyAt"`
    InboundsAdoptedAt   int64     `gorm:"column:inbounds_adopted_at;default:0" json:"-"`
    DownNotified        bool      `gorm:"column:down_notified;default:false" json:"-"`
    CreatedAt           int64     `gorm:"autoCreateTime:milli" json:"createdAt"`
    UpdatedAt           int64     `gorm:"autoUpdateTime:milli" json:"updatedAt"`
}