package constant

import (
	"fmt"
	"ngc-go/packages/compiler/src/output"
)

const (
	constantPrefix = "_c"
	// PoolInclusionLengthThresholdForStrings defines the length threshold for strings
	// Generally all primitive values are excluded from the ConstantPool, but there is an exclusion
	// for strings that reach a certain length threshold.
	PoolInclusionLengthThresholdForStrings = 50
)

// UNKNOWN_VALUE_KEY is used to replace dynamic expressions which can't be safely
// converted into a key. E.g. given an expression `{foo: bar()}`, since we don't know what
// the result of `bar` will be, we create a key that looks like `{foo: <unknown>}`. Note
// that we use a variable, rather than something like `null` in order to avoid collisions.
var UNKNOWN_VALUE_KEY = output.NewReadVarExpr("<unknown>", nil, nil)

// KEY_CONTEXT is the context to use when producing a key.
// This ensures we see the constant not the reference variable when producing a key.
var KEY_CONTEXT = struct{}{}

// FixupExpression is a node that is a place-holder that allows the node to be replaced when the actual
// node is known.
// This allows the constant pool to change an expression from a direct reference to
// a constant to a shared constant. It returns a fix-up node that is later allowed to
// change the referenced expression.
type FixupExpression struct {
	output.ExpressionBase
	original output.OutputExpression
	resolved output.OutputExpression
	shared   bool
}

func NewFixupExpression(resolved output.OutputExpression) *FixupExpression {
	return &FixupExpression{
		ExpressionBase: output.ExpressionBase{
			Type:       resolved.GetType(),
			SourceSpan: resolved.GetSourceSpan(),
		},
		original: resolved,
		resolved: resolved,
		shared:   false,
	}
}

func (f *FixupExpression) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	if context == KEY_CONTEXT {
		// When producing a key we want to traverse the constant not the
		// variable used to refer to it.
		return f.original.VisitExpression(visitor, context)
	}
	return f.resolved.VisitExpression(visitor, context)
}

func (f *FixupExpression) IsEquivalent(e output.OutputExpression) bool {
	if other, ok := e.(*FixupExpression); ok {
		return f.resolved.IsEquivalent(other.resolved)
	}
	return false
}

func (f *FixupExpression) IsConstant() bool {
	return true
}

func (f *FixupExpression) Clone() output.OutputExpression {
	panic("Not supported")
}

func (f *FixupExpression) Fixup(expression output.OutputExpression) {
	f.resolved = expression
	f.shared = true
}

// ConstantPool is a pool of constants that can be reused
type ConstantPool struct {
	statements               []output.OutputStatement
	literals                 map[string]*FixupExpression
	literalFactories         map[string]output.OutputExpression
	sharedConstants          map[string]output.OutputExpression
	claimedNames             map[string]int
	nextNameIndex            int
	isClosureCompilerEnabled bool
}

// NewConstantPool creates a new ConstantPool
func NewConstantPool(isClosureCompilerEnabled bool) *ConstantPool {
	return &ConstantPool{
		statements:               []output.OutputStatement{},
		literals:                 make(map[string]*FixupExpression),
		literalFactories:         make(map[string]output.OutputExpression),
		sharedConstants:          make(map[string]output.OutputExpression),
		claimedNames:             make(map[string]int),
		nextNameIndex:            0,
		isClosureCompilerEnabled: isClosureCompilerEnabled,
	}
}

// GetConstLiteral returns a constant literal, potentially shared
func (cp *ConstantPool) GetConstLiteral(literal output.OutputExpression, forceShared *bool) output.OutputExpression {
	force := forceShared != nil && *forceShared
	if (isLiteralExpr(literal) && !isLongStringLiteral(literal)) || isFixupExpression(literal) {
		// Do not put simple literals into the constant pool or try to produce a constant for a
		// reference to a constant.
		return literal
	}
	key := GenericKeyFnInstance.KeyOf(literal)
	fixup, exists := cp.literals[key]
	newValue := !exists
	if !exists {
		fixup = NewFixupExpression(literal)
		cp.literals[key] = fixup
	}

	if (!newValue && !fixup.shared) || (newValue && force) {
		// Replace the expression with a variable
		name := cp.freshName()
		var value output.OutputExpression
		var usage output.OutputExpression
		if cp.isClosureCompilerEnabled && isLongStringLiteral(literal) {
			// For string literals, Closure will **always** inline the string at
			// **all** usages, duplicating it each time. For large strings, this
			// unnecessarily bloats bundle size. To work around this restriction, we
			// wrap the string in a function, and call that function for each usage.
			// This tricks Closure into using inline logic for functions instead of
			// string literals. Function calls are only inlined if the body is small
			// enough to be worth it. By doing this, very large strings will be
			// shared across multiple usages, rather than duplicating the string at
			// each usage site.
			//
			// const myStr = function() { return "very very very long string"; };
			// const usage1 = myStr();
			// const usage2 = myStr();
			value = output.NewFunctionExpr(
				[]*output.FnParam{}, // Params
				[]output.OutputStatement{
					// Statements
					output.NewReturnStatement(literal, nil, nil),
				},
				nil,
				nil,
				nil,
			)
			usage = output.NewInvokeFunctionExpr(
				output.NewReadVarExpr(name, nil, nil),
				[]output.OutputExpression{},
				nil,
				nil,
				false,
			)
		} else {
			// Just declare and use the variable directly, without a function call
			// indirection. This saves a few bytes and avoids an unnecessary call.
			value = literal
			usage = output.NewReadVarExpr(name, nil, nil)
		}

		cp.statements = append(cp.statements, output.NewDeclareVarStmt(
			name,
			value,
			output.InferredType,
			output.StmtModifierFinal,
			nil,
			nil,
		))
		fixup.Fixup(usage)
	}

	return fixup
}

// GetSharedConstant returns a shared constant expression
func (cp *ConstantPool) GetSharedConstant(def SharedConstantDefinition, expr output.OutputExpression) output.OutputExpression {
	key := def.KeyOf(expr)
	if _, exists := cp.sharedConstants[key]; !exists {
		id := cp.freshName()
		cp.sharedConstants[key] = output.NewReadVarExpr(id, nil, nil)
		cp.statements = append(cp.statements, def.ToSharedConstantDeclaration(id, expr))
	}
	return cp.sharedConstants[key]
}

// GetLiteralFactory returns a literal factory for arrays or maps
func (cp *ConstantPool) GetLiteralFactory(literal output.OutputExpression) (output.OutputExpression, []output.OutputExpression) {
	// Create a pure function that builds an array of a mix of constant and variable expressions
	if arr, ok := literal.(*output.LiteralArrayExpr); ok {
		argumentsForKey := make([]output.OutputExpression, len(arr.Entries))
		for i, e := range arr.Entries {
			if e.IsConstant() {
				argumentsForKey[i] = e
			} else {
				argumentsForKey[i] = UNKNOWN_VALUE_KEY
			}
		}
		key := GenericKeyFnInstance.KeyOf(output.NewLiteralArrayExpr(argumentsForKey, nil, nil))
		return cp.getLiteralFactory(key, arr.Entries, func(entries []output.OutputExpression) output.OutputExpression {
			return output.NewLiteralArrayExpr(entries, nil, nil)
		})
	} else if m, ok := literal.(*output.LiteralMapExpr); ok {
		expressionForKey := output.NewLiteralMapExpr(
			func() []*output.LiteralMapEntry {
				entries := make([]*output.LiteralMapEntry, len(m.Entries))
				for i, e := range m.Entries {
					var value output.OutputExpression
					if e.Value.IsConstant() {
						value = e.Value
					} else {
						value = UNKNOWN_VALUE_KEY
					}
					entries[i] = output.NewLiteralMapEntry(e.Key, value, e.Quoted)
				}
				return entries
			}(),
			nil,
			nil,
		)
		key := GenericKeyFnInstance.KeyOf(expressionForKey)
		return cp.getLiteralFactory(key, func() []output.OutputExpression {
			values := make([]output.OutputExpression, len(m.Entries))
			for i, e := range m.Entries {
				values[i] = e.Value
			}
			return values
		}(), func(entries []output.OutputExpression) output.OutputExpression {
			literalEntries := make([]*output.LiteralMapEntry, len(entries))
			for i, value := range entries {
				literalEntries[i] = output.NewLiteralMapEntry(m.Entries[i].Key, value, m.Entries[i].Quoted)
			}
			return output.NewLiteralMapExpr(literalEntries, nil, nil)
		})
	}
	panic("GetLiteralFactory only supports LiteralArrayExpr and LiteralMapExpr")
}

func (cp *ConstantPool) getLiteralFactory(
	key string,
	values []output.OutputExpression,
	resultMap func([]output.OutputExpression) output.OutputExpression,
) (output.OutputExpression, []output.OutputExpression) {
	literalFactory, exists := cp.literalFactories[key]
	literalFactoryArguments := make([]output.OutputExpression, 0)
	for _, e := range values {
		if !e.IsConstant() {
			literalFactoryArguments = append(literalFactoryArguments, e)
		}
	}
	if !exists {
		resultExpressions := make([]output.OutputExpression, len(values))
		for i, e := range values {
			if e.IsConstant() {
				forceShared := true
				resultExpressions[i] = cp.GetConstLiteral(e, &forceShared)
			} else {
				resultExpressions[i] = output.NewReadVarExpr(fmt.Sprintf("a%d", i), nil, nil)
			}
		}
		parameters := make([]*output.FnParam, 0)
		for _, e := range resultExpressions {
			if isVariable(e) {
				readVar := e.(*output.ReadVarExpr)
				parameters = append(parameters, output.NewFnParam(readVar.Name, output.DynamicType))
			}
		}
		pureFunctionDeclaration := output.NewArrowFunctionExpr(
			parameters,
			resultMap(resultExpressions),
			output.InferredType,
			nil,
		)
		name := cp.freshName()
		cp.statements = append(cp.statements, output.NewDeclareVarStmt(
			name,
			pureFunctionDeclaration,
			output.InferredType,
			output.StmtModifierFinal,
			nil,
			nil,
		))
		literalFactory = output.NewReadVarExpr(name, nil, nil)
		cp.literalFactories[key] = literalFactory
	}
	return literalFactory, literalFactoryArguments
}

// GetSharedFunctionReference returns a shared function reference
func (cp *ConstantPool) GetSharedFunctionReference(fn output.OutputExpression, prefix string, useUniqueName bool) output.OutputExpression {
	isArrow := isArrowFunction(fn)

	for _, current := range cp.statements {
		// Arrow functions are saved as variables so we check if the
		// value of the variable is the same as the arrow function.
		if isArrow {
			if declareVar, ok := current.(*output.DeclareVarStmt); ok && declareVar.Value != nil && declareVar.Value.IsEquivalent(fn) {
				return output.NewReadVarExpr(declareVar.Name, nil, nil)
			}
		}

		// Function declarations are saved as function statements
		// so we compare them directly to the passed-in function.
		if !isArrow {
			if declareFn, ok := current.(*output.DeclareFunctionStmt); ok {
				if fnExpr, ok := fn.(*output.FunctionExpr); ok && fnExpr.IsEquivalentToStmt(declareFn) {
					return output.NewReadVarExpr(declareFn.Name, nil, nil)
				}
			}
		}
	}

	// Otherwise declare the function.
	name := prefix
	if useUniqueName {
		name = cp.UniqueName(prefix, true)
	} else {
		name = cp.UniqueName(prefix, false)
	}
	if fnExpr, ok := fn.(*output.FunctionExpr); ok {
		cp.statements = append(cp.statements, fnExpr.ToDeclStmt(name, output.StmtModifierFinal))
	} else {
		cp.statements = append(cp.statements, output.NewDeclareVarStmt(
			name,
			fn,
			output.InferredType,
			output.StmtModifierFinal,
			fn.GetSourceSpan(),
			nil,
		))
	}
	return output.NewReadVarExpr(name, nil, nil)
}

// UniqueName produces a unique name in the context of this pool.
// The name might be unique among different prefixes if any of the prefixes end in
// a digit so the prefix should be a constant string (not based on user input) and
// must not end in a digit.
func (cp *ConstantPool) UniqueName(name string, alwaysIncludeSuffix bool) string {
	count := cp.claimedNames[name]
	result := name
	if count == 0 && !alwaysIncludeSuffix {
		result = name
	} else {
		result = fmt.Sprintf("%s%d", name, count)
	}
	cp.claimedNames[name] = count + 1
	return result
}

func (cp *ConstantPool) freshName() string {
	return cp.UniqueName(constantPrefix, true)
}

// GetStatements returns all statements in the pool
func (cp *ConstantPool) GetStatements() []output.OutputStatement {
	return cp.statements
}

// AddStatement adds a statement to the pool
func (cp *ConstantPool) AddStatement(stmt output.OutputStatement) {
	cp.statements = append(cp.statements, stmt)
}

// ExpressionKeyFn is an interface for generating keys from expressions
type ExpressionKeyFn interface {
	KeyOf(expr output.OutputExpression) string
}

// SharedConstantDefinition is an interface for shared constant definitions
type SharedConstantDefinition interface {
	ExpressionKeyFn
	ToSharedConstantDeclaration(declName string, keyExpr output.OutputExpression) output.OutputStatement
}

// GenericKeyFn generates keys for expressions
type GenericKeyFn struct{}

var GenericKeyFnInstance = &GenericKeyFn{}

func (g *GenericKeyFn) KeyOf(expr output.OutputExpression) string {
	if lit, ok := expr.(*output.LiteralExpr); ok {
		if str, ok := lit.Value.(string); ok {
			return fmt.Sprintf(`"%s"`, str)
		}
		return fmt.Sprintf("%v", lit.Value)
	} else if regex, ok := expr.(*output.RegularExpressionLiteralExpr); ok {
		flags := ""
		if regex.Flags != nil {
			flags = *regex.Flags
		}
		return fmt.Sprintf("/%s/%s", regex.Body, flags)
	} else if arr, ok := expr.(*output.LiteralArrayExpr); ok {
		entries := make([]string, len(arr.Entries))
		for i, entry := range arr.Entries {
			entries[i] = g.KeyOf(entry)
		}
		return fmt.Sprintf("[%s]", joinStrings(entries, ","))
	} else if m, ok := expr.(*output.LiteralMapExpr); ok {
		entries := make([]string, len(m.Entries))
		for i, entry := range m.Entries {
			key := entry.Key
			if entry.Quoted {
				key = fmt.Sprintf(`"%s"`, key)
			}
			entries[i] = fmt.Sprintf("%s:%s", key, g.KeyOf(entry.Value))
		}
		return fmt.Sprintf("{%s}", joinStrings(entries, ","))
	} else if ext, ok := expr.(*output.ExternalExpr); ok {
		moduleName := "null"
		if ext.Value.ModuleName != nil {
			moduleName = fmt.Sprintf(`"%s"`, *ext.Value.ModuleName)
		}
		name := "null"
		if ext.Value.Name != nil {
			name = fmt.Sprintf(`"%s"`, *ext.Value.Name)
		}
		return fmt.Sprintf("import(%s, %s)", moduleName, name)
	} else if readVar, ok := expr.(*output.ReadVarExpr); ok {
		return fmt.Sprintf("read(%s)", readVar.Name)
	} else if typeof, ok := expr.(*output.TypeofExpr); ok {
		return fmt.Sprintf("typeof(%s)", g.KeyOf(typeof.Expr))
	} else {
		panic(fmt.Sprintf("GenericKeyFn does not handle expressions of type %T", expr))
	}
}

func isVariable(e output.OutputExpression) bool {
	_, ok := e.(*output.ReadVarExpr)
	return ok
}

func isLongStringLiteral(expr output.OutputExpression) bool {
	if lit, ok := expr.(*output.LiteralExpr); ok {
		if str, ok := lit.Value.(string); ok {
			return len(str) >= PoolInclusionLengthThresholdForStrings
		}
	}
	return false
}

func isLiteralExpr(expr output.OutputExpression) bool {
	_, ok := expr.(*output.LiteralExpr)
	return ok
}

func isFixupExpression(expr output.OutputExpression) bool {
	_, ok := expr.(*FixupExpression)
	return ok
}

func isArrowFunction(expr output.OutputExpression) bool {
	_, ok := expr.(*output.ArrowFunctionExpr)
	return ok
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
