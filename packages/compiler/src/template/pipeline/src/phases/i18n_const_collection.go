package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// CollectI18nConsts lifts i18n properties into the consts array.
// This is a placeholder implementation - full implementation should be added later.
func CollectI18nConsts(job *compilation.ComponentCompilationJob) {
	// TODO: Implement full i18n const collection logic
	// This should:
	// - Build lookup maps for i18n contexts, attributes, and expressions
	// - Serialize extracted i18n messages for root i18n blocks and i18n attributes
	// - Serialize I18nAttributes configurations into the const array
	// - Propagate extracted const index into i18n ops
}
