// Package modver compares two versions of the same Go module.
// It can tell whether the differences require at least a patchlevel version change,
// or a minor version change,
// or a major version change,
// according to semver rules
// (https://semver.org/).
package modver
