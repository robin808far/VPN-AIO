package sub

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

var happUserAgentPattern = regexp.MustCompile(`(?i)\bhapp\b`)

type HappConfig struct {
	AutoDetect          bool
	ProviderID          string
	NewURL              string
	FallbackURL         string
	SubInfoColor        string
	SubInfoText         string
	SubInfoButtonText   string
	SubInfoButtonLink   string
	SubExpire           bool
	SubExpireButtonLink string
	NotificationExpire  bool
	NoLimit             bool
	HideSettings        bool
	AlwaysHWID          bool
	TunMode             string
	TunType             string
	ExcludeRoutes       string
	ExcludeAPNs         bool
	ColorProfile        string
	PingType            string
	AutoConnect         bool
	AutoConnectType     string
	PerAppMode          string
	PerAppList          string
	LocalProxyAuth      string
}

func isHappClient(userAgent string) bool {
	return happUserAgentPattern.MatchString(userAgent)
}

func happHeaderValue(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || (unicode.IsControl(r) && r != '\t') {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
}

func setHappBoolHeader(c *gin.Context, key string, enabled bool) {
	if enabled {
		c.Header(key, "1")
		return
	}
	c.Header(key, "0")
}

func applyHappHeaders(c *gin.Context, config HappConfig) {
	if !config.AutoDetect || !isHappClient(c.GetHeader("User-Agent")) {
		return
	}

	for key, value := range map[string]string{
		"ProviderID":     config.ProviderID,
		"New-Url":        config.NewURL,
		"Fallback-Url":   config.FallbackURL,
		"Exclude-Routes": config.ExcludeRoutes,
		"Color-Profile":  config.ColorProfile,
	} {
		if value = happHeaderValue(value); value != "" {
			c.Header(key, value)
		}
	}
	for key, value := range map[string]string{
		"Tun-Mode":  config.TunMode,
		"Tun-Type":  config.TunType,
		"Ping-Type": config.PingType,
	} {
		if value = strings.ToLower(happHeaderValue(value)); value != "" {
			c.Header(key, value)
		}
	}

	infoText := happHeaderValue(config.SubInfoText)
	if infoText == "" {
		c.Header("Sub-Info-Text", "0")
	} else {
		color := strings.ToLower(happHeaderValue(config.SubInfoColor))
		if color != "green" && color != "red" {
			color = "blue"
		}
		c.Header("Sub-Info-Color", color)
		c.Header("Sub-Info-Text", infoText)
		if value := happHeaderValue(config.SubInfoButtonText); value != "" {
			c.Header("Sub-Info-Button-Text", value)
		}
		if value := happHeaderValue(config.SubInfoButtonLink); value != "" {
			c.Header("Sub-Info-Button-Link", value)
		}
	}

	setHappBoolHeader(c, "Sub-Expire", config.SubExpire)
	setHappBoolHeader(c, "Notification-Subs-Expire", config.NotificationExpire)
	setHappBoolHeader(c, "No-Limit-Enabled", config.NoLimit)
	setHappBoolHeader(c, "Hide-Settings", config.HideSettings)
	if config.ExcludeAPNs {
		c.Header("Exclude-Apns-Enable", "true")
	} else {
		c.Header("Exclude-Apns-Enable", "false")
	}
	setHappBoolHeader(c, "Subscription-Always-Hwid-Enable", config.AlwaysHWID)
	setHappBoolHeader(c, "Subscription-Autoconnect", config.AutoConnect)

	if value := happHeaderValue(config.SubExpireButtonLink); value != "" {
		c.Header("Sub-Expire-Button-Link", value)
	}
	if config.AutoConnect {
		autoType := strings.ToLower(happHeaderValue(config.AutoConnectType))
		if autoType != "" {
			c.Header("Subscription-Autoconnect-Type", autoType)
		}
	}
	if mode := strings.ToLower(happHeaderValue(config.LocalProxyAuth)); mode != "" {
		c.Header("Socks-Auth-Mode", mode)
		c.Header("Http-Auth-Mode", mode)
	}
	mode := strings.ToLower(happHeaderValue(config.PerAppMode))
	if mode == "" {
		mode = "off"
	}
	c.Header("Per-App-Proxy-Mode", mode)
	if mode != "off" {
		if list := happHeaderValue(config.PerAppList); list != "" {
			c.Header("Per-App-Proxy-List", list)
		}
	}
}
