package render3

import (
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/facade"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3/r3_identifiers"
)

// R3ConstructorFactoryMetadata contains metadata required by the factory generator
type R3ConstructorFactoryMetadata struct {
	// String name of the type being generated (used to name the factory function)
	Name string

	// An expression representing the interface type being constructed
	Type R3Reference

	// Number of arguments for the `type`
	TypeArgumentCount int

	// Dependencies for the constructor
	// If this is `nil`, then the type's constructor is nonexistent and will be inherited
	// If this is `"invalid"`, then one or more of the parameters wasn't resolvable
	Deps interface{} // []R3DependencyMetadata | "invalid" | nil

	// Type of the target being created by the factory
	Target facade.FactoryTarget
}

// R3FactoryDelegateType represents the type of factory delegate
type R3FactoryDelegateType int

const (
	R3FactoryDelegateTypeClass R3FactoryDelegateType = iota
	R3FactoryDelegateTypeFunction
)

// R3DelegatedFnOrClassMetadata extends R3ConstructorFactoryMetadata with delegate information
type R3DelegatedFnOrClassMetadata struct {
	R3ConstructorFactoryMetadata
	Delegate     output.OutputExpression
	DelegateType R3FactoryDelegateType
	DelegateDeps []R3DependencyMetadata
}

// R3ExpressionFactoryMetadata extends R3ConstructorFactoryMetadata with expression
type R3ExpressionFactoryMetadata struct {
	R3ConstructorFactoryMetadata
	Expression output.OutputExpression
}

// R3DependencyMetadata contains metadata for a dependency
type R3DependencyMetadata struct {
	// An expression representing the token or value to be injected
	// Or `nil` if the dependency could not be resolved - making it invalid
	Token output.OutputExpression

	// If an @Attribute decorator is present, this is the literal type of the attribute name
	// Otherwise it is nil
	AttributeNameType output.OutputExpression

	// Whether the dependency has an @Host qualifier
	Host bool

	// Whether the dependency has an @Optional qualifier
	Optional bool

	// Whether the dependency has an @Self qualifier
	Self bool

	// Whether the dependency has an @SkipSelf qualifier
	SkipSelf bool
}

// CompileFactoryFunction constructs a factory function expression for the given R3FactoryMetadata
func CompileFactoryFunction(meta interface{}) R3CompiledExpression {
	// Type assertion to determine the metadata type
	var ctorMeta *R3ConstructorFactoryMetadata
	var isDelegated bool
	var isExpression bool
	var delegatedMeta *R3DelegatedFnOrClassMetadata
	var expressionMeta *R3ExpressionFactoryMetadata

	if dm, ok := meta.(*R3DelegatedFnOrClassMetadata); ok {
		ctorMeta = &dm.R3ConstructorFactoryMetadata
		isDelegated = true
		delegatedMeta = dm
	} else if em, ok := meta.(*R3ExpressionFactoryMetadata); ok {
		ctorMeta = &em.R3ConstructorFactoryMetadata
		isExpression = true
		expressionMeta = em
	} else if cm, ok := meta.(*R3ConstructorFactoryMetadata); ok {
		ctorMeta = cm
	} else {
		// Fallback - try to extract base metadata
		if m, ok := meta.(interface {
			GetR3ConstructorFactoryMetadata() *R3ConstructorFactoryMetadata
		}); ok {
			ctorMeta = m.GetR3ConstructorFactoryMetadata()
		}
	}

	if ctorMeta == nil {
		panic("Invalid factory metadata")
	}

	t := output.NewReadVarExpr("__ngFactoryType__", nil, nil)
	var baseFactoryVar output.OutputExpression

	// The type to instantiate via constructor invocation
	var typeForCtor output.OutputExpression
	if !isDelegated {
		typeForCtor = output.NewBinaryOperatorExpr(
			output.BinaryOperatorOr,
			t,
			ctorMeta.Type.Value,
			nil,
			nil,
		)
	} else {
		typeForCtor = t
	}

	var ctorExpr output.OutputExpression
	if ctorMeta.Deps != nil {
		// There is a constructor (either explicitly or implicitly defined)
		if deps, ok := ctorMeta.Deps.([]R3DependencyMetadata); ok {
			ctorExpr = output.NewInstantiateExpr(
				typeForCtor,
				injectDependencies(deps, ctorMeta.Target),
				nil,
				nil,
			)
		}
	} else {
		// There is no constructor, use the base class' factory to construct typeForCtor
		baseFactoryVar = output.NewReadVarExpr("ɵ"+ctorMeta.Name+"_BaseFactory", nil, nil)
		ctorExpr = output.NewInvokeFunctionExpr(
			baseFactoryVar,
			[]output.OutputExpression{typeForCtor},
			nil,
			nil,
			false,
		)
	}

	body := []output.OutputStatement{}
	var retExpr output.OutputExpression

	makeConditionalFactory := func(nonCtorExpr output.OutputExpression) output.OutputExpression {
		r := output.NewReadVarExpr("__ngConditionalFactory__", nil, nil)
		body = append(body, output.NewDeclareVarStmt(
			r.Name,
			output.NullExpr,
			nil,
			output.StmtModifierNone,
			nil,
			nil,
		))
		var ctorStmt output.OutputStatement
		if ctorExpr != nil {
			ctorStmt = output.NewExpressionStatement(
				output.NewBinaryOperatorExpr(
					output.BinaryOperatorEquals,
					r,
					ctorExpr,
					nil,
					nil,
				),
				nil,
				nil,
			)
		} else {
			ctorStmt = output.NewExpressionStatement(
				output.NewInvokeFunctionExpr(
					output.NewExternalExpr(r3_identifiers.InvalidFactory, nil, nil, nil),
					[]output.OutputExpression{},
					nil,
					nil,
					false,
				),
				nil,
				nil,
			)
		}
		body = append(body, output.NewIfStmt(
			t,
			[]output.OutputStatement{ctorStmt},
			[]output.OutputStatement{
				output.NewExpressionStatement(
					output.NewBinaryOperatorExpr(
						output.BinaryOperatorEquals,
						r,
						nonCtorExpr,
						nil,
						nil,
					),
					nil,
					nil,
				),
			},
			nil,
			nil,
		))
		return r
	}

	if isDelegated {
		// This type is created with a delegated factory
		delegateArgs := injectDependencies(delegatedMeta.DelegateDeps, ctorMeta.Target)
		var factoryExpr output.OutputExpression
		if delegatedMeta.DelegateType == R3FactoryDelegateTypeClass {
			factoryExpr = output.NewInstantiateExpr(
				delegatedMeta.Delegate,
				delegateArgs,
				nil,
				nil,
			)
		} else {
			factoryExpr = output.NewInvokeFunctionExpr(
				delegatedMeta.Delegate,
				delegateArgs,
				nil,
				nil,
				false,
			)
		}
		retExpr = makeConditionalFactory(factoryExpr)
	} else if isExpression {
		retExpr = makeConditionalFactory(expressionMeta.Expression)
	} else {
		retExpr = ctorExpr
	}

	if retExpr == nil {
		// The expression cannot be formed so render an `ɵɵinvalidFactory()` call
		body = append(body, output.NewExpressionStatement(
			output.NewInvokeFunctionExpr(
				output.NewExternalExpr(r3_identifiers.InvalidFactory, nil, nil, nil),
				[]output.OutputExpression{},
				nil,
				nil,
				false,
			),
			nil,
			nil,
		))
	} else if baseFactoryVar != nil {
		// This factory uses a base factory, so call `ɵɵgetInheritedFactory()` to compute it
		getInheritedFactoryCall := output.NewInvokeFunctionExpr(
			output.NewExternalExpr(r3_identifiers.GetInheritedFactory, nil, nil, nil),
			[]output.OutputExpression{ctorMeta.Type.Value},
			nil,
			nil,
			false,
		)
		// Memoize the base factoryFn: `baseFactory || (baseFactory = ɵɵgetInheritedFactory(...))`
		baseFactory := output.NewBinaryOperatorExpr(
			output.BinaryOperatorOr,
			baseFactoryVar,
			output.NewBinaryOperatorExpr(
				output.BinaryOperatorEquals,
				baseFactoryVar,
				getInheritedFactoryCall,
				nil,
				nil,
			),
			nil,
			nil,
		)
		body = append(body, output.NewReturnStatement(
			output.NewInvokeFunctionExpr(
				baseFactory,
				[]output.OutputExpression{typeForCtor},
				nil,
				nil,
				false,
			),
			nil,
			nil,
		))
	} else {
		// This is straightforward factory, just return it
		body = append(body, output.NewReturnStatement(retExpr, nil, nil))
	}

	factoryName := ctorMeta.Name + "_Factory"
	var factoryFn output.OutputExpression = output.NewFunctionExpr(
		[]*output.FnParam{output.NewFnParam(t.Name, output.DynamicType)},
		body,
		output.InferredType,
		nil,
		&factoryName,
	)

	if baseFactoryVar != nil {
		// There is a base factory variable so wrap its declaration along with the factory function into an IIFE
		baseFactoryVarRead, ok := baseFactoryVar.(*output.ReadVarExpr)
		if !ok {
			panic("baseFactoryVar must be ReadVarExpr")
		}
		factoryFn = output.NewInvokeFunctionExpr(
			output.NewArrowFunctionExpr(
				[]*output.FnParam{},
				[]output.OutputStatement{
					output.NewDeclareVarStmt(baseFactoryVarRead.Name, nil, nil, output.StmtModifierNone, nil, nil),
					output.NewReturnStatement(factoryFn, nil, nil),
				},
				output.InferredType,
				nil,
			),
			[]output.OutputExpression{},
			nil,
			nil,
			true,
		)
	}

	return R3CompiledExpression{
		Expression: factoryFn,
		Statements: []output.OutputStatement{},
		Type:       CreateFactoryType(meta),
	}
}

// CreateFactoryType creates the factory type
func CreateFactoryType(meta interface{}) output.Type {
	var ctorMeta *R3ConstructorFactoryMetadata
	if cm, ok := meta.(*R3ConstructorFactoryMetadata); ok {
		ctorMeta = cm
	} else if dm, ok := meta.(*R3DelegatedFnOrClassMetadata); ok {
		ctorMeta = &dm.R3ConstructorFactoryMetadata
	} else if em, ok := meta.(*R3ExpressionFactoryMetadata); ok {
		ctorMeta = &em.R3ConstructorFactoryMetadata
	}

	if ctorMeta == nil {
		return output.NoneType
	}

	var ctorDepsType output.Type
	if deps, ok := ctorMeta.Deps.([]R3DependencyMetadata); ok {
		ctorDepsType = createCtorDepsType(deps)
	} else {
		ctorDepsType = output.NoneType
	}

	return output.NewExpressionType(
		output.NewExternalExpr(r3_identifiers.FactoryDeclaration, nil, nil, nil),
		output.TypeModifierNone,
		[]output.Type{
			TypeWithParameters(ctorMeta.Type.Type, ctorMeta.TypeArgumentCount),
			ctorDepsType,
		},
	)
}

// injectDependencies injects dependencies
func injectDependencies(deps []R3DependencyMetadata, target facade.FactoryTarget) []output.OutputExpression {
	result := make([]output.OutputExpression, len(deps))
	for i, dep := range deps {
		result[i] = compileInjectDependency(dep, target, i)
	}
	return result
}

// compileInjectDependency compiles a single dependency injection
func compileInjectDependency(
	dep R3DependencyMetadata,
	target facade.FactoryTarget,
	index int,
) output.OutputExpression {
	// Interpret the dependency according to its resolved type
	if dep.Token == nil {
		return output.NewInvokeFunctionExpr(
			output.NewExternalExpr(r3_identifiers.InvalidFactoryDep, nil, nil, nil),
			[]output.OutputExpression{output.NewLiteralExpr(index, nil, nil)},
			nil,
			nil,
			false,
		)
	} else if dep.AttributeNameType == nil {
		// Build up the injection flags according to the metadata
		flags := core.InjectFlagsDefault
		if dep.Self {
			flags |= core.InjectFlagsSelf
		}
		if dep.SkipSelf {
			flags |= core.InjectFlagsSkipSelf
		}
		if dep.Host {
			flags |= core.InjectFlagsHost
		}
		if dep.Optional {
			flags |= core.InjectFlagsOptional
		}
		if target == facade.FactoryTargetPipe {
			flags |= core.InjectFlagsForPipe
		}

		// If this dependency is optional or otherwise has non-default flags, then additional
		// parameters describing how to inject the dependency must be passed to the inject function
		var flagsParam output.OutputExpression
		if flags != core.InjectFlagsDefault || dep.Optional {
			flagsParam = output.NewLiteralExpr(int(flags), nil, nil)
		}

		// Build up the arguments to the injectFn call
		injectArgs := []output.OutputExpression{dep.Token}
		if flagsParam != nil {
			injectArgs = append(injectArgs, flagsParam)
		}
		injectFn := getInjectFn(target)
		return output.NewInvokeFunctionExpr(
			output.NewExternalExpr(&injectFn, nil, nil, nil),
			injectArgs,
			nil,
			nil,
			false,
		)
	} else {
		// The `dep.attributeTypeName` value is defined, which indicates that this is an `@Attribute()`
		// type dependency
		return output.NewInvokeFunctionExpr(
			output.NewExternalExpr(r3_identifiers.InjectAttribute, nil, nil, nil),
			[]output.OutputExpression{dep.Token},
			nil,
			nil,
			false,
		)
	}
}

// createCtorDepsType creates the constructor dependencies type
func createCtorDepsType(deps []R3DependencyMetadata) output.Type {
	hasTypes := false
	attributeTypes := make([]output.OutputExpression, len(deps))
	for i, dep := range deps {
		depType := createCtorDepType(dep)
		if depType != nil {
			hasTypes = true
			attributeTypes[i] = depType
		} else {
			attributeTypes[i] = output.NewLiteralExpr(nil, nil, nil)
		}
	}

	if hasTypes {
		return output.NewExpressionType(
			output.NewLiteralArrayExpr(attributeTypes, nil, nil),
			output.TypeModifierNone,
			nil,
		)
	}
	return output.NoneType
}

// createCtorDepType creates the constructor dependency type
func createCtorDepType(dep R3DependencyMetadata) *output.LiteralMapExpr {
	entries := []*output.LiteralMapEntry{}

	if dep.AttributeNameType != nil {
		entries = append(entries, output.NewLiteralMapEntry(
			"attribute",
			dep.AttributeNameType,
			false,
		))
	}
	if dep.Optional {
		entries = append(entries, output.NewLiteralMapEntry(
			"optional",
			output.NewLiteralExpr(true, nil, nil),
			false,
		))
	}
	if dep.Host {
		entries = append(entries, output.NewLiteralMapEntry(
			"host",
			output.NewLiteralExpr(true, nil, nil),
			false,
		))
	}
	if dep.Self {
		entries = append(entries, output.NewLiteralMapEntry(
			"self",
			output.NewLiteralExpr(true, nil, nil),
			false,
		))
	}
	if dep.SkipSelf {
		entries = append(entries, output.NewLiteralMapEntry(
			"skipSelf",
			output.NewLiteralExpr(true, nil, nil),
			false,
		))
	}

	if len(entries) > 0 {
		return output.NewLiteralMapExpr(entries, nil, nil)
	}
	return nil
}

// IsDelegatedFactoryMetadata checks if metadata is delegated factory metadata
func IsDelegatedFactoryMetadata(meta interface{}) bool {
	_, ok := meta.(*R3DelegatedFnOrClassMetadata)
	return ok
}

// IsExpressionFactoryMetadata checks if metadata is expression factory metadata
func IsExpressionFactoryMetadata(meta interface{}) bool {
	_, ok := meta.(*R3ExpressionFactoryMetadata)
	return ok
}

// getInjectFn gets the inject function for the given target
func getInjectFn(target facade.FactoryTarget) output.ExternalReference {
	switch target {
	case facade.FactoryTargetComponent:
	case facade.FactoryTargetDirective:
	case facade.FactoryTargetPipe:
		return *r3_identifiers.DirectiveInject
	case facade.FactoryTargetNgModule:
	case facade.FactoryTargetInjectable:
	default:
		return *r3_identifiers.Inject
	}
	return *r3_identifiers.Inject
}
