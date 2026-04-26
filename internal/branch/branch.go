package branch

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/forgant-foundry/vergant/internal/config"
)

// Category describes the versioning role of a git branch.
type Category int

const (
	Default Category = iota // trunk branch (main) — minor or major release
	Support                 // support/* — patch release
	Dev                     // dev/* — pre-release targeting next minor (or major)
	Patch                   // patch/* — pre-release targeting next patch
)

// Branch holds the resolved category and metadata for a git branch.
type Branch struct {
	Category    Category
	Name        string
	BuildTicket string // populated for Dev and PatchDev branches
}

// ForName categorizes a branch name using the provided config.
func ForName(cfg *config.Config, name string) (*Branch, error) {
	if name == cfg.DefaultBranch {
		return &Branch{Category: Default, Name: name}, nil
	}

	patchRe, err := regexp.Compile(cfg.SupportBranchRegEx)
	if err != nil {
		return nil, fmt.Errorf("invalid SupportBranchRegEx: %w", err)
	}
	if patchRe.MatchString(name) {
		return &Branch{Category: Support, Name: name}, nil
	}

	devRe, err := regexp.Compile(cfg.DevBranchRegEx)
	if err != nil {
		return nil, fmt.Errorf("invalid DevBranchRegEx: %w", err)
	}
	if m := devRe.FindStringSubmatch(name); m != nil {
		return &Branch{Category: Dev, Name: name, BuildTicket: coerce(m[1])}, nil
	}

	patchDevRe, err := regexp.Compile(cfg.PatchBranchRegEx)
	if err != nil {
		return nil, fmt.Errorf("invalid PatchBranchRegEx: %w", err)
	}
	if m := patchDevRe.FindStringSubmatch(name); m != nil {
		return &Branch{Category: Patch, Name: name, BuildTicket: coerce(m[1])}, nil
	}

	return nil, fmt.Errorf("unsupported branch: %s", name)
}

func coerce(ticket string) string {
	s := strings.ToLower(ticket)
	return regexp.MustCompile(`[-_]`).ReplaceAllLiteralString(s, ".")
}
