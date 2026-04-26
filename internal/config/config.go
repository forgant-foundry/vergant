package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// WorkflowMode controls whether versions are released directly or through candidates first.
type WorkflowMode int

const (
	ReleaseOnly        WorkflowMode = iota
	CandidateToRelease WorkflowMode = iota
)

type file struct {
	MajorVersion     int    `json:"majorVersion"`
	DefaultBranch    string `json:"defaultBranch"`
	PatchBranchRegEx string `json:"patchBranchRegEx"`
	DevBranchRegEx   string `json:"devBranchRegEx"`
	Mode             string `json:"mode"`
}

// Config holds project versioning configuration.
type Config struct {
	MajorVersion     int
	DefaultBranch    string
	PatchBranchRegEx string
	DevBranchRegEx   string
	Mode             WorkflowMode
}

var defaults = Config{
	MajorVersion:     0,
	DefaultBranch:    "main",
	PatchBranchRegEx: `^support\/.*`,
	DevBranchRegEx:   `^.*?\/*(\w+-\d+)\D*`,
	Mode:             ReleaseOnly,
}

// Load reads a config file at path, applying defaults for any missing fields.
// If path is empty or the file does not exist, all defaults are used.
func Load(path string) (*Config, error) {
	if path == "" {
		c := defaults
		return &c, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		c := defaults
		return &c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("unable to access or read config file at %s: %w", path, err)
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("unable to parse config file at %s: %w", path, err)
	}
	c := defaults
	c.MajorVersion = f.MajorVersion
	if f.DefaultBranch != "" {
		c.DefaultBranch = f.DefaultBranch
	}
	if f.PatchBranchRegEx != "" {
		c.PatchBranchRegEx = f.PatchBranchRegEx
	}
	if f.DevBranchRegEx != "" {
		c.DevBranchRegEx = f.DevBranchRegEx
	}
	switch f.Mode {
	case "candidate":
		c.Mode = CandidateToRelease
	case "release":
		c.Mode = ReleaseOnly
	}
	return &c, nil
}
