package render3

import (
	"fmt"
	"ngc-go/packages/compiler/output"
)

const AngularCoreModule = "@angular/core"

// R3JitReflector implements ExternalReferenceResolver which resolves references to @angular/core
// symbols at runtime, according to a consumer-provided mapping.
//
// Only supports ResolveExternalReference, all other methods throw.
type R3JitReflector struct {
	// Context is a map from symbol names to their runtime values
	Context map[string]interface{}
}

// NewR3JitReflector creates a new R3JitReflector with the given context
func NewR3JitReflector(context map[string]interface{}) *R3JitReflector {
	return &R3JitReflector{
		Context: context,
	}
}

// ResolveExternalReference resolves an external reference to its runtime value
// This method implements the ExternalReferenceResolver interface.
// It panics on error to match the TypeScript behavior of throwing errors.
func (r *R3JitReflector) ResolveExternalReference(ref *output.ExternalReference) interface{} {
	// This reflector only handles @angular/core imports
	if ref.ModuleName == nil || *ref.ModuleName != AngularCoreModule {
		moduleName := "<nil>"
		if ref.ModuleName != nil {
			moduleName = *ref.ModuleName
		}
		panic(fmt.Errorf("cannot resolve external reference to %s, only references to %s are supported", moduleName, AngularCoreModule))
	}
	if ref.Name == nil {
		panic(fmt.Errorf("external reference name is nil"))
	}
	if _, ok := r.Context[*ref.Name]; !ok {
		panic(fmt.Errorf("no value provided for %s symbol '%s'", AngularCoreModule, *ref.Name))
	}
	return r.Context[*ref.Name]
}
