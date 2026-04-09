package mobileapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) currentProfile() SessionProfile {
	if s == nil {
		return SessionProfile{}
	}

	path := strings.TrimSpace(s.cfg.ProfileFile)
	if path == "" {
		return s.cfg.Profile
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return s.cfg.Profile
	}

	var profile SessionProfile
	if err := json.Unmarshal(b, &profile); err != nil {
		return s.cfg.Profile
	}

	profile = mergeProfileDefaults(s.cfg.Profile, profile)
	return profile
}

func (s *Server) saveCurrentProfile(profile SessionProfile) error {
	if s == nil {
		return nil
	}

	path := strings.TrimSpace(s.cfg.ProfileFile)
	if path == "" {
		s.cfg.Profile = profile
		return nil
	}

	profile = mergeProfileDefaults(s.cfg.Profile, profile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	b, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	s.cfg.Profile = profile
	return nil
}

func mergeProfileDefaults(base SessionProfile, profile SessionProfile) SessionProfile {
	if strings.TrimSpace(profile.Role) == "" {
		profile.Role = base.Role
	}
	if strings.TrimSpace(profile.DisplayName) == "" {
		profile.DisplayName = base.DisplayName
	}
	if strings.TrimSpace(profile.LegalName) == "" {
		profile.LegalName = base.LegalName
	}
	if strings.TrimSpace(profile.Ref) == "" {
		profile.Ref = base.Ref
	}
	if strings.TrimSpace(profile.Phone) == "" {
		profile.Phone = base.Phone
	}
	if strings.TrimSpace(profile.AvatarURL) == "" {
		profile.AvatarURL = base.AvatarURL
	}
	return profile
}
