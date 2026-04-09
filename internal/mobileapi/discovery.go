package mobileapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
)

const discoveryProbeV1 = "GSCALE_DISCOVER_V1"

type discoveryAnnouncement struct {
	Type        string `json:"type"`
	App         string `json:"app"`
	Service     string `json:"service"`
	ServerName  string `json:"server_name"`
	ServerRef   string `json:"server_ref"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	HTTPPort    int    `json:"http_port"`
}

func (s *Server) ListenAndServeDiscovery(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp4", strings.TrimSpace(s.cfg.DiscoveryAddr))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, 2048)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		if strings.TrimSpace(string(buf[:n])) != discoveryProbeV1 {
			continue
		}
		profile := s.currentProfile()
		payload, err := json.Marshal(discoveryAnnouncement{
			Type:        "gscale_announce_v1",
			App:         "gscale-zebra",
			Service:     "mobileapi",
			ServerName:  s.cfg.ServerName,
			ServerRef:   profile.Ref,
			DisplayName: profile.DisplayName,
			Role:        profile.Role,
			HTTPPort:    httpPortFromListenAddr(s.cfg.ListenAddr),
		})
		if err != nil {
			return err
		}
		if _, err := conn.WriteToUDP(payload, remote); err != nil && ctx.Err() != nil {
			return nil
		}
	}
}

func httpPortFromListenAddr(addr string) int {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 8081
	}
	if strings.HasPrefix(addr, ":") {
		port, err := strconv.Atoi(strings.TrimPrefix(addr, ":"))
		if err == nil && port > 0 {
			return port
		}
		return 8081
	}

	_, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return 8081
	}
	port, err := strconv.Atoi(strings.TrimSpace(portText))
	if err != nil || port <= 0 {
		return 8081
	}
	return port
}
