// Package strategy implements the two version calculation strategies that sit
// between the git layer and the tool facade.
//
// # AcquireLastVersion
//
// [AcquireLastVersion.Resolve] returns the last known version for a branch,
// using the branch category to choose which query to issue:
//
//   - Default and patch branches: the most recent release or candidate tag
//     reachable from HEAD ([Repository.LastVersion]).
//   - Dev branches: the most recent dev tag for the branch's ticket, falling
//     back to the last release/candidate if no dev tag for that ticket exists
//     yet ([Repository.LastVersionForDevelopment]). The fallback ensures the
//     first dev tag on a branch is anchored to the correct base version.
//
// # NewVersionCalculator
//
// [NewVersionCalculator.Resolve] calculates the next version given a branch and
// the last known version:
//
//   - Default branch: increment minor. If the last version's major is below the
//     configured majorVersion, return a new major.0.0 instead. If it is above,
//     return an error — the config is out of sync with history.
//   - Patch branch: increment patch. Returns an error if no prior version exists,
//     since a patch branch with no history to patch against is invalid.
//   - Dev branch: if no prior version, start a new build from the configured
//     major. If the prior version has no build metadata, start a new build
//     series for the ticket. If it already has build metadata, increment the
//     trailing counter.
//
// Both types accept a [git.Repository] interface and are fully exercisable in
// unit tests without a real repository.
package strategy
