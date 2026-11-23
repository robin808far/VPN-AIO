package model

// ApiToken is a bearer token for machine-to-machine API access. The plaintext
// is shown once at creation; only its SHA-256 hash is stored. Scopes is a
// comma-separated list (currently "node-sync", "node-admin") checked per route.
type ApiToken struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name       string `json:"name" gorm:"unique"`
	TokenHash  string `json:"-" gorm:"uniqueIndex;size:64"`
	Scopes     string `json:"scopes"`
	CreatedAt  int64  `json:"createdAt" gorm:"autoCreateTime"`
	LastUsedAt int64  `json:"lastUsedAt"`
}

// Node is a remote VPN-AIO panel registered on this master panel. The token is
// the remote panel's API token (scope node-admin there) presented as a Bearer
// credential on every call the master makes to it.
type Node struct {
	Id        int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" form:"name" gorm:"unique"`
	Scheme    string `json:"scheme" form:"scheme" gorm:"default:http"`
	Address   string `json:"address" form:"address"`
	Port      int    `json:"port" form:"port"`
	BasePath  string `json:"basePath" form:"basePath" gorm:"default:/"`
	Token     string `json:"-" form:"token"`
	SyncMode  string `json:"syncMode" form:"syncMode" gorm:"default:all"`
	Enabled   bool   `json:"enabled" form:"enabled" gorm:"default:true"`
	Status    string `json:"status" form:"status" gorm:"default:unknown"`
	Version   string `json:"version" form:"version"`
	Inbounds  int    `json:"inbounds" form:"inbounds"`
	LastSeen  int64  `json:"lastSeen" form:"lastSeen"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt int64  `json:"updatedAt" gorm:"autoUpdateTime"`
}
