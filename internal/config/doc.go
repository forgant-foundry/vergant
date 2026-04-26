// Package config loads vergant's project-level configuration from a YAML file.
//
// # Configuration file
//
// The file is optional. All fields have defaults; a missing file is equivalent
// to an empty file. The default filename expected by the CLI is
// .vergant.yml in the working directory.
//
// The file is flat key-value YAML — one field per line, no nesting. Lines
// beginning with # are comments; blank lines are ignored. Values may optionally
// be enclosed in single or double quotes.
//
//	majorVersion: 1
//	defaultBranch: main
//	patchBranchRegEx: ^support\/.*
//	devBranchRegEx: ^.*?\/*(\w+-\d+)\D*
//	mode: release
//
// majorVersion controls the major component of generated versions. When the
// last reachable version has a lower major, vergant generates major.0.0. A
// config major below the last reachable major is an error — it indicates the
// config is out of sync with the tag history.
//
// mode accepts "release" (default) or "candidate". In release mode, default
// and patch branches produce r-prefixed tags directly. In candidate mode they
// produce c-prefixed tags that must be explicitly promoted to r via the
// promote command.
//
// patchBranchRegEx and devBranchRegEx are standard Go regular expressions.
// devBranchRegEx must contain exactly one capture group whose match becomes
// the ticket identifier embedded in dev version build metadata.
//
// # Loading
//
// [Load] reads the file at path and merges it over defaults. A missing file
// returns defaults without error. An empty path returns defaults. A file that
// exists but contains invalid YAML returns an error.
package config
