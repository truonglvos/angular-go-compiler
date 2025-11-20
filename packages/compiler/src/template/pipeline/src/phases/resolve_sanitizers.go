package phases

import (
	"strings"

	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/output"
	r3_identifiers "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/schema"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"
	pipeline_util "ngc-go/packages/compiler/src/template/pipeline/src/util"

	"ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// sanitizerFns maps security contexts to their sanitizer function
var sanitizerFns = map[core.SecurityContext]*output.ExternalReference{
	core.SecurityContextHTML:         r3_identifiers.SanitizeHtml,
	core.SecurityContextRESOURCE_URL: r3_identifiers.SanitizeResourceUrl,
	core.SecurityContextSCRIPT:       r3_identifiers.SanitizeScript,
	core.SecurityContextSTYLE:        r3_identifiers.SanitizeStyle,
	core.SecurityContextURL:          r3_identifiers.SanitizeUrl,
}

// trustedValueFns maps security contexts to their trusted value function
var trustedValueFns = map[core.SecurityContext]*output.ExternalReference{
	core.SecurityContextHTML:         r3_identifiers.TrustConstantHtml,
	core.SecurityContextRESOURCE_URL: r3_identifiers.TrustConstantResourceUrl,
}

// ResolveSanitizers resolves sanitization functions for ops that need them.
func ResolveSanitizers(job *compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		elements := pipeline_util.CreateOpXrefMap(unit)

		// For normal element bindings we create trusted values for security sensitive constant
		// attributes. However, for host bindings we skip this step (this matches what
		// TemplateDefinitionBuilder does).
		// TODO: Is the TDB behavior correct here?
		if job.Kind != compilation.CompilationJobKindHost {
			for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
				if extractedAttr, ok := op.(*ops_create.ExtractedAttributeOp); ok {
					securityCtx := getOnlySecurityContext(extractedAttr.SecurityContext)
					trustedValueFn := trustedValueFns[securityCtx]
					if trustedValueFn != nil {
						extractedAttr.TrustedValueFn = output.NewExternalExpr(trustedValueFn, nil, nil, nil)
					} else {
						extractedAttr.TrustedValueFn = nil
					}
				}
			}
		}

		for op := unit.GetUpdate().Head(); op != nil; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindProperty, ir.OpKindAttribute, ir.OpKindDomProperty:
				var sanitizerFn *output.ExternalReference
				var propertyOp *ops_update.PropertyOp
				var attributeOp *ops_update.AttributeOp
				var domPropertyOp *ops_update.DomPropertyOp
				var securityCtx interface{} // core.SecurityContext | []core.SecurityContext

				if prop, ok := op.(*ops_update.PropertyOp); ok {
					propertyOp = prop
					securityCtx = prop.SecurityContext
				} else if attr, ok := op.(*ops_update.AttributeOp); ok {
					attributeOp = attr
					securityCtx = attr.SecurityContext
				} else if domProp, ok := op.(*ops_update.DomPropertyOp); ok {
					domPropertyOp = domProp
					securityCtx = domProp.SecurityContext
				}

				if securityCtxArray, ok := securityCtx.([]core.SecurityContext); ok {
					// When the host element isn't known, some URL attributes (such as "src" and "href") may
					// be part of multiple different security contexts. In this case we use special
					// sanitization function and select the actual sanitizer at runtime based on a tag name
					// that is provided while invoking sanitization function.
					if len(securityCtxArray) == 2 {
						hasURL := false
						hasResourceURL := false
						for _, ctx := range securityCtxArray {
							if ctx == core.SecurityContextURL {
								hasURL = true
							}
							if ctx == core.SecurityContextRESOURCE_URL {
								hasResourceURL = true
							}
						}
						if hasURL && hasResourceURL {
							sanitizerFn = r3_identifiers.SanitizeUrlOrResourceUrl
						} else {
							sanitizerFn = sanitizerFns[getOnlySecurityContext(securityCtx)]
						}
					} else {
						sanitizerFn = sanitizerFns[getOnlySecurityContext(securityCtx)]
					}
				} else {
					sanitizerFn = sanitizerFns[getOnlySecurityContext(securityCtx)]
				}

				var opName string
				if propertyOp != nil {
					opName = propertyOp.Name
					if sanitizerFn != nil {
						propertyOp.Sanitizer = output.NewExternalExpr(sanitizerFn, nil, nil, nil)
					} else {
						propertyOp.Sanitizer = nil
					}
				} else if attributeOp != nil {
					opName = attributeOp.Name
					if sanitizerFn != nil {
						attributeOp.Sanitizer = output.NewExternalExpr(sanitizerFn, nil, nil, nil)
					} else {
						attributeOp.Sanitizer = nil
					}
				} else if domPropertyOp != nil {
					opName = domPropertyOp.Name
					if sanitizerFn != nil {
						domPropertyOp.Sanitizer = output.NewExternalExpr(sanitizerFn, nil, nil, nil)
					} else {
						domPropertyOp.Sanitizer = nil
					}
				}

				// If there was no sanitization function found based on the security context of an
				// attribute/property, check whether this attribute/property is one of the
				// security-sensitive <iframe> attributes (and that the current element is actually an
				// <iframe>).
				if sanitizerFn == nil {
					var isIframe bool
					if job.Kind == compilation.CompilationJobKindHost || op.GetKind() == ir.OpKindDomProperty {
						// Note: for host bindings defined on a directive, we do not try to find all
						// possible places where it can be matched, so we can not determine whether
						// the host element is an <iframe>. In this case, we just assume it is and append a
						// validation function, which is invoked at runtime and would have access to the
						// underlying DOM element to check if it's an <iframe> and if so - run extra checks.
						isIframe = true
					} else {
						// For a normal binding we can just check if the element its on is an iframe.
						var targetXref ir_operations.XrefId
						if propertyOp != nil {
							targetXref = propertyOp.Target
						} else if attributeOp != nil {
							targetXref = attributeOp.Target
						} else if domPropertyOp != nil {
							targetXref = domPropertyOp.Target
						}

						ownerOp, ok := elements[targetXref]
						if !ok || !ops_create.IsElementOrContainerOp(ownerOp) {
							panic("Property should have an element-like owner")
						}
						isIframe = isIframeElement(ownerOp)
					}

					if isIframe && schema.IsIframeSecuritySensitiveAttr(opName) {
						validateFn := r3_identifiers.ValidateIframeAttribute
						if propertyOp != nil {
							propertyOp.Sanitizer = output.NewExternalExpr(validateFn, nil, nil, nil)
						} else if attributeOp != nil {
							attributeOp.Sanitizer = output.NewExternalExpr(validateFn, nil, nil, nil)
						} else if domPropertyOp != nil {
							domPropertyOp.Sanitizer = output.NewExternalExpr(validateFn, nil, nil, nil)
						}
					}
				}
			}
		}
	}
}

// isIframeElement checks whether the given op represents an iframe element.
func isIframeElement(op ir_operations.CreateOp) bool {
	if elementStart, ok := op.(*ops_create.ElementStartOp); ok {
		if elementStart.Tag != nil {
			return strings.ToLower(*elementStart.Tag) == "iframe"
		}
	}
	return false
}

// getOnlySecurityContext asserts that there is only a single security context and returns it.
func getOnlySecurityContext(securityContext interface{}) core.SecurityContext {
	if ctxArray, ok := securityContext.([]core.SecurityContext); ok {
		if len(ctxArray) > 1 {
			// TODO: What should we do here? TDB just took the first one, but this feels like something we
			// would want to know about and create a special case for like we did for Url/ResourceUrl. My
			// guess is that, outside of the Url/ResourceUrl case, this never actually happens. If there
			// do turn out to be other cases, throwing an error until we can address it feels safer.
			panic("AssertionError: Ambiguous security context")
		}
		if len(ctxArray) == 0 {
			return core.SecurityContextNONE
		}
		return ctxArray[0]
	}
	if ctx, ok := securityContext.(core.SecurityContext); ok {
		return ctx
	}
	return core.SecurityContextNONE
}
