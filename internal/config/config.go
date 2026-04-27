package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/forgant-foundry/vergant/internal/version"
)

// WorkflowMode controls whether versions are released directly or through candidates first.
type WorkflowMode int

const (
	ReleaseOnly        WorkflowMode = iota
	CandidateToRelease WorkflowMode = iota
)

// Config holds project versioning configuration.
type Config struct {
	MajorVersion       int
	DefaultBranch      string
	SupportBranchRegEx string
	DevBranchRegEx     string
	PatchBranchRegEx   string
	Mode               WorkflowMode
	ReleasePrefix      string // tag prefix for releases; defaults to "v"
	CandidatePrefix    string // tag prefix for candidates; defaults to "c"
	DevPrefix          string // tag prefix for dev builds; defaults to "d"
}

// Prefixes returns the configured prefix strings, applying defaults for any empty fields.
func (c *Config) Prefixes() version.Prefixes {
	p := version.DefaultPrefixes()
	if c.ReleasePrefix != "" {
		p.Release = c.ReleasePrefix
	}
	if c.CandidatePrefix != "" {
		p.Candidate = c.CandidatePrefix
	}
	if c.DevPrefix != "" {
		p.Dev = c.DevPrefix
	}
	return p
}

var defaults = Config{
	MajorVersion:        0,
	DefaultBranch:       "main",
	SupportBranchRegEx:    `^support\/.*`,
	DevBranchRegEx:      `^dev\/(.+)$`,
	PatchBranchRegEx: `^patch\/(.+)$`,
	Mode:                ReleaseOnly,
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
	return parse(path, data)
}

func parse(path string, data []byte) (*Config, error) {
	c := defaults
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			return nil, fmt.Errorf("unable to parse config file at %s: line %d: missing colon separator", path, lineNum)
		}
		key := strings.TrimSpace(line[:idx])
		value := unquote(strings.TrimSpace(line[idx+1:]))
		switch key {
		case "majorVersion":
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("unable to parse config file at %s: line %d: majorVersion must be an integer", path, lineNum)
			}
			c.MajorVersion = n
		case "defaultBranch":
			if value != "" {
				c.DefaultBranch = value
			}
		case "supportBranchRegEx":
			if value != "" {
				c.SupportBranchRegEx = value
			}
		case "devBranchRegEx":
			if value != "" {
				c.DevBranchRegEx = value
			}
		case "patchBranchRegEx":
			if value != "" {
				c.PatchBranchRegEx = value
			}
		case "mode":
			switch value {
			case "candidate":
				c.Mode = CandidateToRelease
			case "release":
				c.Mode = ReleaseOnly
			}
		case "releasePrefix":
			if value != "" {
				c.ReleasePrefix = value
			}
		case "candidatePrefix":
			if value != "" {
				c.CandidatePrefix = value
			}
		case "devPrefix":
			if value != "" {
				c.DevPrefix = value
			}
		}
	}
	return &c, nil
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
