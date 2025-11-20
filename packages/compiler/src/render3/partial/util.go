package partial

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/render3/view"
)

// ToOptionalLiteralArray creates an array literal expression from the given array, mapping all values to an expression
// using the provided mapping function. If the array is empty or null, then null is returned.
//
// @param values The array to transfer into literal array expression.
// @param mapper The logic to use for creating an expression for the array's values.
// @returns An array literal expression representing `values`, or null if `values` is empty or
// is itself null.
func ToOptionalLiteralArray[T any](
	values []T,
	mapper func(value T) output.OutputExpression,
) *output.LiteralArrayExpr {
	if values == nil || len(values) == 0 {
		return nil
	}
	literals := make([]output.OutputExpression, len(values))
	for i, value := range values {
		literals[i] = mapper(value)
	}
	return output.NewLiteralArrayExpr(literals, nil, nil)
}

// ToOptionalLiteralMap creates an object literal expression from the given object, mapping all values to an expression
// using the provided mapping function. If the object has no keys, then null is returned.
//
// @param object The object to transfer into an object literal expression.
// @param mapper The logic to use for creating an expression for the object's values.
// @returns An object literal expression representing `object`, or null if `object` does not have
// any keys.
func ToOptionalLiteralMap[T any](
	object map[string]T,
	mapper func(value T) output.OutputExpression,
) *output.LiteralMapExpr {
	if object == nil || len(object) == 0 {
		return nil
	}
	entries := make([]*output.LiteralMapEntry, 0, len(object))
	for key, value := range object {
		entries = append(entries, output.NewLiteralMapEntry(key, mapper(value), true))
	}
	if len(entries) > 0 {
		return output.NewLiteralMapExpr(entries, nil, nil)
	}
	return nil
}

// CompileDependencies compiles dependencies into a literal expression
func CompileDependencies(
	deps interface{}, // []render3.R3DependencyMetadata | "invalid" | nil
) output.OutputExpression {
	if depsStr, ok := deps.(string); ok && depsStr == "invalid" {
		// The `deps` can be set to the string "invalid"  by the `unwrapConstructorDependencies()`
		// function, which tries to convert `ConstructorDeps` into `R3DependencyMetadata[]`.
		return output.NewLiteralExpr("invalid", output.InferredType, nil)
	} else if deps == nil {
		return output.NewLiteralExpr(nil, output.InferredType, nil)
	} else if depsArray, ok := deps.([]render3.R3DependencyMetadata); ok {
		literals := make([]output.OutputExpression, len(depsArray))
		for i, dep := range depsArray {
			literals[i] = CompileDependency(dep)
		}
		return output.NewLiteralArrayExpr(literals, nil, nil)
	}
	panic("Invalid deps type")
}

// CompileDependency compiles a single dependency into a literal map expression
func CompileDependency(dep render3.R3DependencyMetadata) *output.LiteralMapExpr {
	depMeta := view.NewDefinitionMap()
	depMeta.Set("token", dep.Token)
	if dep.AttributeNameType != nil {
		depMeta.Set("attribute", output.NewLiteralExpr(true, output.InferredType, nil))
	}
	if dep.Host {
		depMeta.Set("host", output.NewLiteralExpr(true, output.InferredType, nil))
	}
	if dep.Optional {
		depMeta.Set("optional", output.NewLiteralExpr(true, output.InferredType, nil))
	}
	if dep.Self {
		depMeta.Set("self", output.NewLiteralExpr(true, output.InferredType, nil))
	}
	if dep.SkipSelf {
		depMeta.Set("skipSelf", output.NewLiteralExpr(true, output.InferredType, nil))
	}
	return depMeta.ToLiteralMap()
}
