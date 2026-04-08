package mobileapi

import (
	"os"
	"strings"
)

const (
	defaultListenAddr      = ":8081"
	defaultBridgeStateFile = "/tmp/gscale-zebra/bridge_state.json"
	defaultPolygonURL      = "http://127.0.0.1:18000"
)

type Config struct {
	ListenAddr      string
	BridgeStateFile string
	PolygonURL      string
	LoginPhone      string
	LoginCode       string
	Profile         SessionProfile
}

func LoadConfig() Config {
	role := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		os.Getenv("MOBILE_API_ROLE"),
		"admin",
	)))
	if role == "" {
		role = "admin"
	}

	phone := firstNonEmpty(os.Getenv("MOBILE_API_PHONE"), "998900000000")
	displayName := firstNonEmpty(os.Getenv("MOBILE_API_DISPLAY_NAME"), "Polygon Operator")
	legalName := firstNonEmpty(os.Getenv("MOBILE_API_LEGAL_NAME"), displayName)
	ref := firstNonEmpty(os.Getenv("MOBILE_API_REF"), "dev-operator")
	avatarURL := strings.TrimSpace(os.Getenv("MOBILE_API_AVATAR_URL"))

	return Config{
		ListenAddr:      firstNonEmpty(os.Getenv("MOBILE_API_ADDR"), defaultListenAddr),
		BridgeStateFile: firstNonEmpty(os.Getenv("BRIDGE_STATE_FILE"), defaultBridgeStateFile),
		PolygonURL:      firstNonEmpty(os.Getenv("POLYGON_URL"), defaultPolygonURL),
		LoginPhone:      phone,
		LoginCode:       firstNonEmpty(os.Getenv("MOBILE_API_CODE"), "1234"),
		Profile: SessionProfile{
			Role:        role,
			DisplayName: displayName,
			LegalName:   legalName,
			Ref:         ref,
			Phone:       phone,
			AvatarURL:   avatarURL,
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
