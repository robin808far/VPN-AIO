package sub

import (
	"crypto/ecdh"
	"encoding/base64"
	"strings"
)

func singboxStringList(value any) []string {
	switch value := value.(type) {
	case string:
		return []string{value}
	case []string:
		return value
	case []any:
		var result []string
		for _, item := range value {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	}
	return nil
}

// Only client-facing values are exported; the server's REALITY private key never
// leaves the panel. Missing/invalid mandatory export data refuses the link.
func singboxShareParams(block map[string]any) (map[string]string, bool) {
	params := map[string]string{"security": "none", "type": "tcp"}
	tls, _ := block["tls"].(map[string]any)
	if enabled, _ := tls["enabled"].(bool); enabled {
		params["security"] = "tls"
		params["sni"], _ = tls["server_name"].(string)
		params["alpn"] = strings.Join(singboxStringList(tls["alpn"]), ",")
		if reality, ok := tls["reality"].(map[string]any); ok {
			if enabled, _ := reality["enabled"].(bool); enabled {
				private, _ := reality["private_key"].(string)
				raw, err := base64.RawURLEncoding.DecodeString(private)
				if err != nil {
					return nil, false
				}
				key, err := ecdh.X25519().NewPrivateKey(raw)
				if err != nil {
					return nil, false
				}
				params["security"] = "reality"
				params["pbk"] = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
				params["fp"] = "chrome"
				ids := singboxStringList(reality["short_id"])
				if len(ids) > 0 {
					params["sid"] = ids[0]
				}
			}
		}
	}
	transport, _ := block["transport"].(map[string]any)
	if transport == nil {
		return params, true
	}
	typ, _ := transport["type"].(string)
	params["type"] = typ
	switch typ {
	case "ws", "http", "httpupgrade":
		params["path"], _ = transport["path"].(string)
		hosts := singboxStringList(transport["host"])
		if len(hosts) > 0 {
			params["host"] = hosts[0]
		}
		if headers, ok := transport["headers"].(map[string]any); ok {
			for key, value := range headers {
				if strings.EqualFold(key, "host") && params["host"] == "" {
					values := singboxStringList(value)
					if len(values) > 0 {
						params["host"] = values[0]
					}
				}
			}
		}
	case "grpc":
		params["serviceName"], _ = transport["service_name"].(string)
	default:
		return nil, false
	}
	return params, true
}
