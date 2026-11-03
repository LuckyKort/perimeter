// Package kordon holds files from the repository root that the app embeds.
package kordon

import _ "embed"

// Changelog is CHANGELOG.md: the app shows its own version's section after an update.
//
//go:embed CHANGELOG.md
var Changelog string
