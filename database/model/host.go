package model

// Host is one advertised endpoint on one inbound; GroupId joins rows edited together.
// Adapted from MHSanaei/3x-ui's managed Hosts design under the project's GPL-3.0 license.
type Host struct {
	Id                     int      `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupId                string   `json:"groupId" gorm:"column:group_id;index"`
	InboundId              int      `json:"inboundId" gorm:"column:inbound_id;index;not null"`
	SortOrder              int      `json:"sortOrder" gorm:"column:sort_order"`
	Remark                 string   `json:"remark"`
	ServerDescription      string   `json:"serverDescription"`
	IsDisabled             bool     `json:"isDisabled"`
	Tags                   []string `json:"tags" gorm:"serializer:json"`
	Address                string   `json:"address"`
	Port                   int      `json:"port"`
	Security               string   `json:"security"`
	Sni                    string   `json:"sni"`
	Snis                   []string `json:"snis" gorm:"serializer:json"`
	SniSharedRandom        bool     `json:"sniSharedRandom"`
	HostHeader             string   `json:"hostHeader"`
	Path                   string   `json:"path"`
	Alpn                   []string `json:"alpn" gorm:"serializer:json"`
	Fingerprint            string   `json:"fingerprint"`
	CipherSuites           string   `json:"cipherSuites"`
	OverrideSniFromAddress bool     `json:"overrideSniFromAddress"`
	KeepSniBlank           bool     `json:"keepSniBlank"`
	PinnedPeerCertSha256   []string `json:"pinnedPeerCertSha256" gorm:"serializer:json"`
	VerifyPeerCertByName   string   `json:"verifyPeerCertByName"`
	AllowInsecure          bool     `json:"allowInsecure"`
	EchConfigList          string   `json:"echConfigList"`
	MuxParams              string   `json:"muxParams" gorm:"type:text"`
	SockoptParams          string   `json:"sockoptParams" gorm:"type:text"`
	FinalMask              string   `json:"finalMask" gorm:"type:text"`
	XhttpExtraParams       string   `json:"xhttpExtraParams" gorm:"type:text"`
	VlessRoute             string   `json:"vlessRoute"`
	ExcludeFromSubTypes    []string `json:"excludeFromSubTypes" gorm:"serializer:json"`
	MihomoIpVersion        string   `json:"mihomoIpVersion"`
	MihomoX25519           bool     `json:"mihomoX25519"`
	RandomCount            int      `json:"randomCount"`
	LegacySource           string   `json:"-" gorm:"index"`
	CreatedAt              int64    `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt              int64    `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

type HostGroup struct {
	Host
	InboundIds []int    `json:"inboundIds"`
	Hosts      []string `json:"hosts"`
}

// Deleted bindings retain their inbound scope instead of exposing another endpoint.
type HostTombstone struct {
	HostId    int `json:"hostId" gorm:"primaryKey;autoIncrement:false"`
	InboundId int `json:"inboundId" gorm:"index"`
}
