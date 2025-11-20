package pipeline_instruction

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	r3_identifiers "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_shared "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"
	"ngc-go/packages/compiler/src/util"
)

// elementOrContainerBase is a helper function for creating element or container operations
func elementOrContainerBase(
	instruction output.ExternalReference,
	slot int,
	tag *string,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
	}
	if tag != nil {
		args = append(args, output.NewLiteralExpr(*tag, nil, nil))
	}
	if localRefIndex != nil {
		var constIdx interface{} = nil
		if constIndex != nil {
			constIdx = *constIndex
		}
		args = append(args,
			output.NewLiteralExpr(constIdx, nil, nil),
			output.NewLiteralExpr(*localRefIndex, nil, nil),
		)
	} else if constIndex != nil {
		args = append(args, output.NewLiteralExpr(*constIndex, nil, nil))
	}

	return call(instruction, args, sourceSpan)
}

// Element creates an element operation
func Element(
	slot int,
	tag string,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return elementOrContainerBase(
		*r3_identifiers.Element,
		slot,
		&tag,
		constIndex,
		localRefIndex,
		sourceSpan,
	)
}

// ElementStart creates an element start operation
func ElementStart(
	slot int,
	tag string,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return elementOrContainerBase(
		*r3_identifiers.ElementStart,
		slot,
		&tag,
		constIndex,
		localRefIndex,
		sourceSpan,
	)
}

// ElementEnd creates an element end operation
func ElementEnd(sourceSpan *util.ParseSourceSpan) ir_operations.CreateOp {
	return call(*r3_identifiers.ElementEnd, []output.OutputExpression{}, sourceSpan)
}

// ElementContainerStart creates an element container start operation
func ElementContainerStart(
	slot int,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return elementOrContainerBase(
		*r3_identifiers.ElementContainerStart,
		slot,
		nil,
		constIndex,
		localRefIndex,
		sourceSpan,
	)
}

// ElementContainer creates an element container operation
func ElementContainer(
	slot int,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return elementOrContainerBase(
		*r3_identifiers.ElementContainer,
		slot,
		nil,
		constIndex,
		localRefIndex,
		sourceSpan,
	)
}

// ElementContainerEnd creates an element container end operation
func ElementContainerEnd() ir_operations.CreateOp {
	return call(*r3_identifiers.ElementContainerEnd, []output.OutputExpression{}, nil)
}

// templateBase is a helper function for creating template operations
func templateBase(
	instruction output.ExternalReference,
	slot int,
	templateFnRef output.OutputExpression,
	decls int,
	vars int,
	tag *string,
	constIndex *int,
	localRefs *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
		templateFnRef,
		output.NewLiteralExpr(decls, nil, nil),
		output.NewLiteralExpr(vars, nil, nil),
	}
	if tag != nil {
		args = append(args, output.NewLiteralExpr(*tag, nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if constIndex != nil {
		args = append(args, output.NewLiteralExpr(*constIndex, nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if localRefs != nil {
		args = append(args, output.NewLiteralExpr(*localRefs, nil, nil))
		args = append(args, output.NewExternalExpr(r3_identifiers.TemplateRefExtractor, nil, nil, nil))
	}

	// Remove trailing null expressions
	for len(args) > 0 {
		last := args[len(args)-1]
		if lit, ok := last.(*output.LiteralExpr); ok && lit.Value == nil {
			args = args[:len(args)-1]
		} else {
			break
		}
	}

	return call(instruction, args, sourceSpan)
}

// Template creates a template operation
func Template(
	slot int,
	templateFnRef output.OutputExpression,
	decls int,
	vars int,
	tag *string,
	constIndex *int,
	localRefs *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return templateBase(
		*r3_identifiers.TemplateCreate,
		slot,
		templateFnRef,
		decls,
		vars,
		tag,
		constIndex,
		localRefs,
		sourceSpan,
	)
}

// DisableBindings creates a disable bindings operation
func DisableBindings() ir_operations.CreateOp {
	return call(*r3_identifiers.DisableBindings, []output.OutputExpression{}, nil)
}

// EnableBindings creates an enable bindings operation
func EnableBindings() ir_operations.CreateOp {
	return call(*r3_identifiers.EnableBindings, []output.OutputExpression{}, nil)
}

// propertyBase is a helper function for creating property operations
func propertyBase(
	instruction output.ExternalReference,
	name string,
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sanitizer output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(name, nil, nil),
	}

	var expr output.OutputExpression
	if interp, ok := expression.(*ops_update.Interpolation); ok {
		expr = interpolationToExpression(interp, sourceSpan)
	} else if e, ok := expression.(output.OutputExpression); ok {
		expr = e
	} else {
		panic(fmt.Sprintf("invalid expression type: %T", expression))
	}
	args = append(args, expr)

	if sanitizer != nil {
		args = append(args, sanitizer)
	}

	return call(instruction, args, sourceSpan)
}

// Listener creates a listener operation
func Listener(
	name string,
	handlerFn output.OutputExpression,
	eventTargetResolver *output.ExternalReference,
	syntheticHost bool,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(name, nil, nil),
		handlerFn,
	}
	if eventTargetResolver != nil {
		args = append(args, output.NewExternalExpr(eventTargetResolver, nil, nil, nil))
	}
	identifier := r3_identifiers.Listener
	if syntheticHost {
		identifier = r3_identifiers.SyntheticHostListener
	}
	return call(*identifier, args, sourceSpan)
}

// TwoWayBindingSet creates a two-way binding set expression
func TwoWayBindingSet(target output.OutputExpression, value output.OutputExpression) output.OutputExpression {
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.TwoWayBindingSet, nil, nil, nil),
		[]output.OutputExpression{target, value},
		nil,
		nil,
		false,
	)
}

// TwoWayListener creates a two-way listener operation
func TwoWayListener(
	name string,
	handlerFn output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return call(
		*r3_identifiers.TwoWayListener,
		[]output.OutputExpression{
			output.NewLiteralExpr(name, nil, nil),
			handlerFn,
		},
		sourceSpan,
	)
}

// Pipe creates a pipe operation
func Pipe(slot int, name string) ir_operations.CreateOp {
	return call(
		*r3_identifiers.Pipe,
		[]output.OutputExpression{
			output.NewLiteralExpr(slot, nil, nil),
			output.NewLiteralExpr(name, nil, nil),
		},
		nil,
	)
}

// NamespaceHTML creates a namespace HTML operation
func NamespaceHTML() ir_operations.CreateOp {
	return call(*r3_identifiers.NamespaceHTML, []output.OutputExpression{}, nil)
}

// NamespaceSVG creates a namespace SVG operation
func NamespaceSVG() ir_operations.CreateOp {
	return call(*r3_identifiers.NamespaceSVG, []output.OutputExpression{}, nil)
}

// NamespaceMath creates a namespace MathML operation
func NamespaceMath() ir_operations.CreateOp {
	return call(*r3_identifiers.NamespaceMathML, []output.OutputExpression{}, nil)
}

// Advance creates an advance operation
func Advance(delta int, sourceSpan *util.ParseSourceSpan) ir_operations.UpdateOp {
	args := []output.OutputExpression{}
	if delta > 1 {
		args = append(args, output.NewLiteralExpr(delta, nil, nil))
	}
	return call(*r3_identifiers.Advance, args, sourceSpan)
}

// Reference creates a reference expression
func Reference(slot int) output.OutputExpression {
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.Reference, nil, nil, nil),
		[]output.OutputExpression{output.NewLiteralExpr(slot, nil, nil)},
		nil,
		nil,
		false,
	)
}

// NextContext creates a next context expression
func NextContext(steps int) output.OutputExpression {
	args := []output.OutputExpression{}
	if steps != 1 {
		args = append(args, output.NewLiteralExpr(steps, nil, nil))
	}
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.NextContext, nil, nil, nil),
		args,
		nil,
		nil,
		false,
	)
}

// GetCurrentView creates a get current view expression
func GetCurrentView() output.OutputExpression {
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.GetCurrentView, nil, nil, nil),
		[]output.OutputExpression{},
		nil,
		nil,
		false,
	)
}

// RestoreView creates a restore view expression
func RestoreView(savedView output.OutputExpression) output.OutputExpression {
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.RestoreView, nil, nil, nil),
		[]output.OutputExpression{savedView},
		nil,
		nil,
		false,
	)
}

// ResetView creates a reset view expression
func ResetView(returnValue output.OutputExpression) output.OutputExpression {
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.ResetView, nil, nil, nil),
		[]output.OutputExpression{returnValue},
		nil,
		nil,
		false,
	)
}

// Text creates a text operation
func Text(
	slot int,
	initialValue string,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
	}
	if initialValue != "" {
		args = append(args, output.NewLiteralExpr(initialValue, nil, nil))
	}
	return call(*r3_identifiers.Text, args, sourceSpan)
}

// call is a helper function to create a statement operation from an instruction call
func call(
	instruction output.ExternalReference,
	args []output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	expr := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(&instruction, nil, nil, nil),
		args,
		nil,
		sourceSpan,
		false,
	)
	stmt := output.NewExpressionStatement(expr, sourceSpan, nil)
	return ops_shared.NewStatementOp(stmt)
}

// Property creates a property operation
func Property(
	name string,
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sanitizer output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	return propertyBase(*r3_identifiers.Property, name, expression, sanitizer, sourceSpan)
}

// AriaProperty creates an aria property operation
func AriaProperty(
	name string,
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	return propertyBase(*r3_identifiers.AriaProperty, name, expression, nil, sourceSpan)
}

// DomProperty creates a DOM property operation
func DomProperty(
	name string,
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sanitizer output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	return propertyBase(*r3_identifiers.DomProperty, name, expression, sanitizer, sourceSpan)
}

// Control creates a control operation
func Control(
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sanitizer output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	args := []output.OutputExpression{}
	var expr output.OutputExpression
	if interp, ok := expression.(*ops_update.Interpolation); ok {
		expr = interpolationToExpression(interp, sourceSpan)
	} else if e, ok := expression.(output.OutputExpression); ok {
		expr = e
	} else {
		panic(fmt.Sprintf("invalid expression type: %T", expression))
	}
	args = append(args, expr)
	if sanitizer != nil {
		args = append(args, sanitizer)
	}
	return call(*r3_identifiers.Control, args, sourceSpan)
}

// ControlCreate creates a control create operation
func ControlCreate(sourceSpan *util.ParseSourceSpan) ir_operations.CreateOp {
	return call(*r3_identifiers.ControlCreate, []output.OutputExpression{}, sourceSpan)
}

// TwoWayProperty creates a two-way property operation
func TwoWayProperty(
	name string,
	expression output.OutputExpression,
	sanitizer output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(name, nil, nil),
		expression,
	}
	if sanitizer != nil {
		args = append(args, sanitizer)
	}
	return call(*r3_identifiers.TwoWayProperty, args, sourceSpan)
}

// Attribute creates an attribute operation
func Attribute(
	name string,
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sanitizer output.OutputExpression,
	namespace *string,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(name, nil, nil),
	}

	var expr output.OutputExpression
	if interp, ok := expression.(*ops_update.Interpolation); ok {
		expr = interpolationToExpression(interp, sourceSpan)
	} else if e, ok := expression.(output.OutputExpression); ok {
		expr = e
	} else {
		panic(fmt.Sprintf("invalid expression type: %T", expression))
	}
	args = append(args, expr)

	if sanitizer != nil || namespace != nil {
		if sanitizer != nil {
			args = append(args, sanitizer)
		} else {
			args = append(args, output.NewLiteralExpr(nil, nil, nil))
		}
	}
	if namespace != nil {
		args = append(args, output.NewLiteralExpr(*namespace, nil, nil))
	}

	return call(*r3_identifiers.Attribute, args, nil)
}

// StyleProp creates a style property operation
func StyleProp(
	name string,
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	unit *string,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(name, nil, nil),
	}

	var expr output.OutputExpression
	if interp, ok := expression.(*ops_update.Interpolation); ok {
		expr = interpolationToExpression(interp, sourceSpan)
	} else if e, ok := expression.(output.OutputExpression); ok {
		expr = e
	} else {
		panic(fmt.Sprintf("invalid expression type: %T", expression))
	}
	args = append(args, expr)

	if unit != nil {
		args = append(args, output.NewLiteralExpr(*unit, nil, nil))
	}

	return call(*r3_identifiers.StyleProp, args, sourceSpan)
}

// ClassProp creates a class property operation
func ClassProp(
	name string,
	expression output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	return call(
		*r3_identifiers.ClassProp,
		[]output.OutputExpression{
			output.NewLiteralExpr(name, nil, nil),
			expression,
		},
		sourceSpan,
	)
}

// StyleMap creates a style map operation
func StyleMap(
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	var value output.OutputExpression
	if interp, ok := expression.(*ops_update.Interpolation); ok {
		value = interpolationToExpression(interp, sourceSpan)
	} else if e, ok := expression.(output.OutputExpression); ok {
		value = e
	} else {
		panic(fmt.Sprintf("invalid expression type: %T", expression))
	}
	return call(*r3_identifiers.StyleMap, []output.OutputExpression{value}, sourceSpan)
}

// ClassMap creates a class map operation
func ClassMap(
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	var value output.OutputExpression
	if interp, ok := expression.(*ops_update.Interpolation); ok {
		value = interpolationToExpression(interp, sourceSpan)
	} else if e, ok := expression.(output.OutputExpression); ok {
		value = e
	} else {
		panic(fmt.Sprintf("invalid expression type: %T", expression))
	}
	return call(*r3_identifiers.ClassMap, []output.OutputExpression{value}, sourceSpan)
}

// collateInterpolationArgs collates string and expression arguments for an interpolation instruction
func collateInterpolationArgs(strings []string, expressions []output.OutputExpression) []output.OutputExpression {
	if len(strings) < 1 || len(expressions) != len(strings)-1 {
		panic(fmt.Sprintf(
			"expected specific shape of args for strings/expressions in interpolation: strings=%d, expressions=%d",
			len(strings),
			len(expressions),
		))
	}

	interpolationArgs := []output.OutputExpression{}

	if len(expressions) == 1 && strings[0] == "" && strings[1] == "" {
		interpolationArgs = append(interpolationArgs, expressions[0])
	} else {
		for idx := 0; idx < len(expressions); idx++ {
			interpolationArgs = append(interpolationArgs,
				output.NewLiteralExpr(strings[idx], nil, nil),
				expressions[idx],
			)
		}
		// idx points at the last string
		interpolationArgs = append(interpolationArgs, output.NewLiteralExpr(strings[len(expressions)], nil, nil))
	}

	return interpolationArgs
}

// interpolationToExpression converts an interpolation to an expression
func interpolationToExpression(
	interpolation *ops_update.Interpolation,
	sourceSpan *util.ParseSourceSpan,
) output.OutputExpression {
	interpolationArgs := collateInterpolationArgs(interpolation.Strings, interpolation.Expressions)
	return callVariadicInstructionExpr(
		ValueInterpolateConfig,
		[]output.OutputExpression{},
		interpolationArgs,
		[]output.OutputExpression{},
		sourceSpan,
	)
}

// TextInterpolate creates a text interpolate operation
func TextInterpolate(
	strings []string,
	expressions []output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	interpolationArgs := collateInterpolationArgs(strings, expressions)
	return callVariadicInstruction(TextInterpolateConfig, []output.OutputExpression{}, interpolationArgs, []output.OutputExpression{}, sourceSpan)
}

// VariadicInstructionConfig describes a specific flavor of instruction used to represent variadic instructions
type VariadicInstructionConfig struct {
	Constant []output.ExternalReference
	Variable *output.ExternalReference
	Mapping  func(argCount int) int
}

// TextInterpolateConfig is the config for the textInterpolate instruction
var TextInterpolateConfig = VariadicInstructionConfig{
	Constant: []output.ExternalReference{
		*r3_identifiers.TextInterpolate,
		*r3_identifiers.TextInterpolate1,
		*r3_identifiers.TextInterpolate2,
		*r3_identifiers.TextInterpolate3,
		*r3_identifiers.TextInterpolate4,
		*r3_identifiers.TextInterpolate5,
		*r3_identifiers.TextInterpolate6,
		*r3_identifiers.TextInterpolate7,
		*r3_identifiers.TextInterpolate8,
	},
	Variable: r3_identifiers.TextInterpolateV,
	Mapping: func(n int) int {
		if n%2 == 0 {
			panic("expected odd number of arguments")
		}
		return (n - 1) / 2
	},
}

// ValueInterpolateConfig is the config for the value interpolate instruction
var ValueInterpolateConfig = VariadicInstructionConfig{
	Constant: []output.ExternalReference{
		*r3_identifiers.Interpolate,
		*r3_identifiers.Interpolate1,
		*r3_identifiers.Interpolate2,
		*r3_identifiers.Interpolate3,
		*r3_identifiers.Interpolate4,
		*r3_identifiers.Interpolate5,
		*r3_identifiers.Interpolate6,
		*r3_identifiers.Interpolate7,
		*r3_identifiers.Interpolate8,
	},
	Variable: r3_identifiers.InterpolateV,
	Mapping: func(n int) int {
		if n%2 == 0 {
			panic("expected odd number of arguments")
		}
		return (n - 1) / 2
	},
}

// PureFunctionConfig is the config for the pure function instruction
var PureFunctionConfig = VariadicInstructionConfig{
	Constant: []output.ExternalReference{
		*r3_identifiers.PureFunction0,
		*r3_identifiers.PureFunction1,
		*r3_identifiers.PureFunction2,
		*r3_identifiers.PureFunction3,
		*r3_identifiers.PureFunction4,
		*r3_identifiers.PureFunction5,
		*r3_identifiers.PureFunction6,
		*r3_identifiers.PureFunction7,
		*r3_identifiers.PureFunction8,
	},
	Variable: r3_identifiers.PureFunctionV,
	Mapping: func(n int) int {
		return n
	},
}

// callVariadicInstructionExpr calls a variadic instruction and returns an expression
func callVariadicInstructionExpr(
	config VariadicInstructionConfig,
	baseArgs []output.OutputExpression,
	interpolationArgs []output.OutputExpression,
	extraArgs []output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) output.OutputExpression {
	// mapping need to be done before potentially dropping the last interpolation argument
	n := config.Mapping(len(interpolationArgs))

	// In the case the interpolation instruction ends with an empty string we drop it
	// And the runtime will take care of it.
	if len(interpolationArgs) > 0 {
		lastInterpolationArg := interpolationArgs[len(interpolationArgs)-1]
		if len(extraArgs) == 0 &&
			len(interpolationArgs) > 1 {
			if lit, ok := lastInterpolationArg.(*output.LiteralExpr); ok {
				if str, ok := lit.Value.(string); ok && str == "" {
					interpolationArgs = interpolationArgs[:len(interpolationArgs)-1]
				}
			}
		}
	}

	if n < len(config.Constant) {
		// Constant calling pattern
		allArgs := append(baseArgs, interpolationArgs...)
		allArgs = append(allArgs, extraArgs...)
		return output.NewInvokeFunctionExpr(
			output.NewExternalExpr(&config.Constant[n], nil, nil, nil),
			allArgs,
			nil,
			sourceSpan,
			false,
		)
	} else if config.Variable != nil {
		// Variable calling pattern
		allArgs := append(baseArgs, output.NewLiteralArrayExpr(interpolationArgs, nil, nil))
		allArgs = append(allArgs, extraArgs...)
		return output.NewInvokeFunctionExpr(
			output.NewExternalExpr(config.Variable, nil, nil, nil),
			allArgs,
			nil,
			sourceSpan,
			false,
		)
	} else {
		panic("unable to call variadic function")
	}
}

// callVariadicInstruction calls a variadic instruction and returns an update operation
func callVariadicInstruction(
	config VariadicInstructionConfig,
	baseArgs []output.OutputExpression,
	interpolationArgs []output.OutputExpression,
	extraArgs []output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	expr := callVariadicInstructionExpr(config, baseArgs, interpolationArgs, extraArgs, sourceSpan)
	stmt := output.NewExpressionStatement(expr, sourceSpan, nil)
	return ops_shared.NewStatementOp(stmt)
}

// Defer creates a defer operation
func Defer(
	selfSlot int,
	primarySlot int,
	dependencyResolverFn output.OutputExpression,
	loadingSlot *int,
	placeholderSlot *int,
	errorSlot *int,
	loadingConfig output.OutputExpression,
	placeholderConfig output.OutputExpression,
	enableTimerScheduling bool,
	sourceSpan *util.ParseSourceSpan,
	flags *ir.TDeferDetailsFlags,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(selfSlot, nil, nil),
		output.NewLiteralExpr(primarySlot, nil, nil),
	}
	if dependencyResolverFn != nil {
		args = append(args, dependencyResolverFn)
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if loadingSlot != nil {
		args = append(args, output.NewLiteralExpr(*loadingSlot, nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if placeholderSlot != nil {
		args = append(args, output.NewLiteralExpr(*placeholderSlot, nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if errorSlot != nil {
		args = append(args, output.NewLiteralExpr(*errorSlot, nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if loadingConfig != nil {
		args = append(args, loadingConfig)
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if placeholderConfig != nil {
		args = append(args, placeholderConfig)
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if enableTimerScheduling {
		args = append(args, output.NewExternalExpr(r3_identifiers.DeferEnableTimerScheduling, nil, nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if flags != nil {
		args = append(args, output.NewLiteralExpr(int(*flags), nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}

	// Remove trailing null expressions
	for len(args) > 0 {
		last := args[len(args)-1]
		if lit, ok := last.(*output.LiteralExpr); ok && lit.Value == nil {
			args = args[:len(args)-1]
		} else {
			break
		}
	}

	return call(*r3_identifiers.Defer, args, sourceSpan)
}

// deferTriggerToR3TriggerInstructionsMap maps defer trigger kinds to their instruction identifiers
var deferTriggerToR3TriggerInstructionsMap = map[ir.DeferTriggerKind]map[ir.DeferOpModifierKind]output.ExternalReference{
	ir.DeferTriggerKindIdle: {
		ir.DeferOpModifierKindNone:     *r3_identifiers.DeferOnIdle,
		ir.DeferOpModifierKindPrefetch: *r3_identifiers.DeferPrefetchOnIdle,
		ir.DeferOpModifierKindHydrate:  *r3_identifiers.DeferHydrateOnIdle,
	},
	ir.DeferTriggerKindImmediate: {
		ir.DeferOpModifierKindNone:     *r3_identifiers.DeferOnImmediate,
		ir.DeferOpModifierKindPrefetch: *r3_identifiers.DeferPrefetchOnImmediate,
		ir.DeferOpModifierKindHydrate:  *r3_identifiers.DeferHydrateOnImmediate,
	},
	ir.DeferTriggerKindTimer: {
		ir.DeferOpModifierKindNone:     *r3_identifiers.DeferOnTimer,
		ir.DeferOpModifierKindPrefetch: *r3_identifiers.DeferPrefetchOnTimer,
		ir.DeferOpModifierKindHydrate:  *r3_identifiers.DeferHydrateOnTimer,
	},
	ir.DeferTriggerKindHover: {
		ir.DeferOpModifierKindNone:     *r3_identifiers.DeferOnHover,
		ir.DeferOpModifierKindPrefetch: *r3_identifiers.DeferPrefetchOnHover,
		ir.DeferOpModifierKindHydrate:  *r3_identifiers.DeferHydrateOnHover,
	},
	ir.DeferTriggerKindInteraction: {
		ir.DeferOpModifierKindNone:     *r3_identifiers.DeferOnInteraction,
		ir.DeferOpModifierKindPrefetch: *r3_identifiers.DeferPrefetchOnInteraction,
		ir.DeferOpModifierKindHydrate:  *r3_identifiers.DeferHydrateOnInteraction,
	},
	ir.DeferTriggerKindViewport: {
		ir.DeferOpModifierKindNone:     *r3_identifiers.DeferOnViewport,
		ir.DeferOpModifierKindPrefetch: *r3_identifiers.DeferPrefetchOnViewport,
		ir.DeferOpModifierKindHydrate:  *r3_identifiers.DeferHydrateOnViewport,
	},
	ir.DeferTriggerKindNever: {
		ir.DeferOpModifierKindNone:     *r3_identifiers.DeferHydrateNever,
		ir.DeferOpModifierKindPrefetch: *r3_identifiers.DeferHydrateNever,
		ir.DeferOpModifierKindHydrate:  *r3_identifiers.DeferHydrateNever,
	},
}

// DeferOn creates a defer on operation
func DeferOn(
	trigger ir.DeferTriggerKind,
	args []output.OutputExpression,
	modifier ir.DeferOpModifierKind,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	triggerMap, ok := deferTriggerToR3TriggerInstructionsMap[trigger]
	if !ok {
		panic(fmt.Sprintf("unable to determine instruction for trigger %v", trigger))
	}
	instructionToCall, ok := triggerMap[modifier]
	if !ok {
		panic(fmt.Sprintf("unable to determine instruction for trigger %v with modifier %v", trigger, modifier))
	}
	return call(instructionToCall, args, sourceSpan)
}

// ProjectionDef creates a projection definition operation
func ProjectionDef(def output.OutputExpression) ir_operations.CreateOp {
	args := []output.OutputExpression{}
	if def != nil {
		args = append(args, def)
	}
	return call(*r3_identifiers.ProjectionDef, args, nil)
}

// Projection creates a projection operation
func Projection(
	slot int,
	projectionSlotIndex int,
	attributes *output.LiteralArrayExpr,
	fallbackFnName *string,
	fallbackDecls *int,
	fallbackVars *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
	}
	if projectionSlotIndex != 0 || attributes != nil || fallbackFnName != nil {
		args = append(args, output.NewLiteralExpr(projectionSlotIndex, nil, nil))
		if attributes != nil {
			args = append(args, attributes)
		}
		if fallbackFnName != nil {
			if attributes == nil {
				args = append(args, output.NewLiteralExpr(nil, nil, nil))
			}
			args = append(args,
				output.NewReadVarExpr(*fallbackFnName, nil, nil),
				output.NewLiteralExpr(*fallbackDecls, nil, nil),
				output.NewLiteralExpr(*fallbackVars, nil, nil),
			)
		}
	}
	return call(*r3_identifiers.Projection, args, sourceSpan)
}

// I18nStart creates an i18n start operation
func I18nStart(
	slot int,
	constIndex int,
	subTemplateIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
		output.NewLiteralExpr(constIndex, nil, nil),
	}
	if subTemplateIndex != nil {
		args = append(args, output.NewLiteralExpr(*subTemplateIndex, nil, nil))
	}
	return call(*r3_identifiers.I18nStart, args, sourceSpan)
}

// ConditionalCreate creates a conditional create operation
func ConditionalCreate(
	slot int,
	templateFnRef output.OutputExpression,
	decls int,
	vars int,
	tag *string,
	constIndex *int,
	localRefs *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return templateBase(
		*r3_identifiers.ConditionalCreate,
		slot,
		templateFnRef,
		decls,
		vars,
		tag,
		constIndex,
		localRefs,
		sourceSpan,
	)
}

// ConditionalBranchCreate creates a conditional branch create operation
func ConditionalBranchCreate(
	slot int,
	templateFnRef output.OutputExpression,
	decls int,
	vars int,
	tag *string,
	constIndex *int,
	localRefs *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return templateBase(
		*r3_identifiers.ConditionalBranchCreate,
		slot,
		templateFnRef,
		decls,
		vars,
		tag,
		constIndex,
		localRefs,
		sourceSpan,
	)
}

// RepeaterCreate creates a repeater create operation
func RepeaterCreate(
	slot int,
	viewFnName string,
	decls int,
	vars int,
	tag *string,
	constIndex *int,
	trackByFn output.OutputExpression,
	trackByUsesComponentInstance bool,
	emptyViewFnName *string,
	emptyDecls *int,
	emptyVars *int,
	emptyTag *string,
	emptyConstIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
		output.NewReadVarExpr(viewFnName, nil, nil),
		output.NewLiteralExpr(decls, nil, nil),
		output.NewLiteralExpr(vars, nil, nil),
	}
	if tag != nil {
		args = append(args, output.NewLiteralExpr(*tag, nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	if constIndex != nil {
		args = append(args, output.NewLiteralExpr(*constIndex, nil, nil))
	} else {
		args = append(args, output.NewLiteralExpr(nil, nil, nil))
	}
	args = append(args, trackByFn)
	if trackByUsesComponentInstance || emptyViewFnName != nil {
		args = append(args, output.NewLiteralExpr(trackByUsesComponentInstance, nil, nil))
		if emptyViewFnName != nil {
			args = append(args,
				output.NewReadVarExpr(*emptyViewFnName, nil, nil),
				output.NewLiteralExpr(*emptyDecls, nil, nil),
				output.NewLiteralExpr(*emptyVars, nil, nil),
			)
			if emptyTag != nil || emptyConstIndex != nil {
				if emptyTag != nil {
					args = append(args, output.NewLiteralExpr(*emptyTag, nil, nil))
				} else {
					args = append(args, output.NewLiteralExpr(nil, nil, nil))
				}
			}
			if emptyConstIndex != nil {
				args = append(args, output.NewLiteralExpr(*emptyConstIndex, nil, nil))
			}
		}
	}
	return call(*r3_identifiers.RepeaterCreate, args, sourceSpan)
}

// Repeater creates a repeater operation
func Repeater(
	collection output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	return call(*r3_identifiers.Repeater, []output.OutputExpression{collection}, sourceSpan)
}

// DeferWhen creates a defer when operation
func DeferWhen(
	modifier ir.DeferOpModifierKind,
	expr output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	if modifier == ir.DeferOpModifierKindPrefetch {
		return call(*r3_identifiers.DeferPrefetchWhen, []output.OutputExpression{expr}, sourceSpan)
	} else if modifier == ir.DeferOpModifierKindHydrate {
		return call(*r3_identifiers.DeferHydrateWhen, []output.OutputExpression{expr}, sourceSpan)
	}
	return call(*r3_identifiers.DeferWhen, []output.OutputExpression{expr}, sourceSpan)
}

// DeclareLet creates a declare let operation
func DeclareLet(slot int, sourceSpan *util.ParseSourceSpan) ir_operations.CreateOp {
	return call(*r3_identifiers.DeclareLet, []output.OutputExpression{output.NewLiteralExpr(slot, nil, nil)}, sourceSpan)
}

// StoreLet creates a store let expression
func StoreLet(value output.OutputExpression, sourceSpan *util.ParseSourceSpan) output.OutputExpression {
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.StoreLet, nil, nil, nil),
		[]output.OutputExpression{value},
		nil,
		sourceSpan,
		false,
	)
}

// ReadContextLet creates a read context let expression
func ReadContextLet(slot int) output.OutputExpression {
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.ReadContextLet, nil, nil, nil),
		[]output.OutputExpression{output.NewLiteralExpr(slot, nil, nil)},
		nil,
		nil,
		false,
	)
}

// I18n creates an i18n operation
func I18n(
	slot int,
	constIndex int,
	subTemplateIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
		output.NewLiteralExpr(constIndex, nil, nil),
	}
	if subTemplateIndex != nil {
		args = append(args, output.NewLiteralExpr(*subTemplateIndex, nil, nil))
	}
	return call(*r3_identifiers.I18n, args, sourceSpan)
}

// I18nEnd creates an i18n end operation
func I18nEnd(endSourceSpan *util.ParseSourceSpan) ir_operations.CreateOp {
	return call(*r3_identifiers.I18nEnd, []output.OutputExpression{}, endSourceSpan)
}

// I18nAttributes creates an i18n attributes operation
func I18nAttributes(slot int, i18nAttributesConfig int) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
		output.NewLiteralExpr(i18nAttributesConfig, nil, nil),
	}
	return call(*r3_identifiers.I18nAttributes, args, nil)
}

// I18nExp creates an i18n expression operation
func I18nExp(expr output.OutputExpression, sourceSpan *util.ParseSourceSpan) ir_operations.UpdateOp {
	return call(*r3_identifiers.I18nExp, []output.OutputExpression{expr}, sourceSpan)
}

// I18nApply creates an i18n apply operation
func I18nApply(slot int, sourceSpan *util.ParseSourceSpan) ir_operations.UpdateOp {
	return call(*r3_identifiers.I18nApply, []output.OutputExpression{output.NewLiteralExpr(slot, nil, nil)}, sourceSpan)
}

// Conditional creates a conditional operation
func Conditional(
	condition output.OutputExpression,
	contextValue output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	args := []output.OutputExpression{condition}
	if contextValue != nil {
		args = append(args, contextValue)
	}
	return call(*r3_identifiers.Conditional, args, sourceSpan)
}

// PureFunction creates a pure function expression
func PureFunction(
	varOffset int,
	fn output.OutputExpression,
	args []output.OutputExpression,
) output.OutputExpression {
	return callVariadicInstructionExpr(
		PureFunctionConfig,
		[]output.OutputExpression{
			output.NewLiteralExpr(varOffset, nil, nil),
			fn,
		},
		args,
		[]output.OutputExpression{},
		nil,
	)
}

// AttachSourceLocation creates an attach source location operation
func AttachSourceLocation(
	templatePath string,
	locations *output.LiteralArrayExpr,
) ir_operations.CreateOp {
	return call(
		*r3_identifiers.AttachSourceLocations,
		[]output.OutputExpression{
			output.NewLiteralExpr(templatePath, nil, nil),
			locations,
		},
		nil,
	)
}

// DomElement creates a DOM element operation
func DomElement(
	slot int,
	tag string,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return elementOrContainerBase(
		*r3_identifiers.DomElement,
		slot,
		&tag,
		constIndex,
		localRefIndex,
		sourceSpan,
	)
}

// DomElementStart creates a DOM element start operation
func DomElementStart(
	slot int,
	tag string,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return elementOrContainerBase(
		*r3_identifiers.DomElementStart,
		slot,
		&tag,
		constIndex,
		localRefIndex,
		sourceSpan,
	)
}

// DomElementEnd creates a DOM element end operation
func DomElementEnd(sourceSpan *util.ParseSourceSpan) ir_operations.CreateOp {
	return call(*r3_identifiers.DomElementEnd, []output.OutputExpression{}, sourceSpan)
}

// DomElementContainerStart creates a DOM element container start operation
func DomElementContainerStart(
	slot int,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return elementOrContainerBase(
		*r3_identifiers.DomElementContainerStart,
		slot,
		nil,
		constIndex,
		localRefIndex,
		sourceSpan,
	)
}

// DomElementContainer creates a DOM element container operation
func DomElementContainer(
	slot int,
	constIndex *int,
	localRefIndex *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return elementOrContainerBase(
		*r3_identifiers.DomElementContainer,
		slot,
		nil,
		constIndex,
		localRefIndex,
		sourceSpan,
	)
}

// DomElementContainerEnd creates a DOM element container end operation
func DomElementContainerEnd() ir_operations.CreateOp {
	return call(*r3_identifiers.DomElementContainerEnd, []output.OutputExpression{}, nil)
}

// DomListener creates a DOM listener operation
func DomListener(
	name string,
	handlerFn output.OutputExpression,
	eventTargetResolver *output.ExternalReference,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{
		output.NewLiteralExpr(name, nil, nil),
		handlerFn,
	}
	if eventTargetResolver != nil {
		args = append(args, output.NewExternalExpr(eventTargetResolver, nil, nil, nil))
	}
	return call(*r3_identifiers.DomListener, args, sourceSpan)
}

// DomTemplate creates a DOM template operation
func DomTemplate(
	slot int,
	templateFnRef output.OutputExpression,
	decls int,
	vars int,
	tag *string,
	constIndex *int,
	localRefs *int,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	return templateBase(
		*r3_identifiers.DomTemplate,
		slot,
		templateFnRef,
		decls,
		vars,
		tag,
		constIndex,
		localRefs,
		sourceSpan,
	)
}

// PipeBindings contains pipe binding identifiers
var PipeBindings = []output.ExternalReference{
	*r3_identifiers.PipeBind1,
	*r3_identifiers.PipeBind2,
	*r3_identifiers.PipeBind3,
	*r3_identifiers.PipeBind4,
}

// PipeBind creates a pipe bind expression
func PipeBind(slot int, varOffset int, args []output.OutputExpression) output.OutputExpression {
	if len(args) < 1 || len(args) > len(PipeBindings) {
		panic(fmt.Sprintf("pipeBind() argument count out of bounds: %d", len(args)))
	}

	instruction := PipeBindings[len(args)-1]
	allArgs := []output.OutputExpression{
		output.NewLiteralExpr(slot, nil, nil),
		output.NewLiteralExpr(varOffset, nil, nil),
	}
	allArgs = append(allArgs, args...)
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(&instruction, nil, nil, nil),
		allArgs,
		nil,
		nil,
		false,
	)
}

// PipeBindV creates a variadic pipe bind expression
func PipeBindV(slot int, varOffset int, args output.OutputExpression) output.OutputExpression {
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.PipeBindV, nil, nil, nil),
		[]output.OutputExpression{
			output.NewLiteralExpr(slot, nil, nil),
			output.NewLiteralExpr(varOffset, nil, nil),
			args,
		},
		nil,
		nil,
		false,
	)
}

// Animation creates an animation operation
func Animation(
	animationKind ir.AnimationKind,
	handlerFn output.OutputExpression,
	sanitizer output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{handlerFn}
	if sanitizer != nil {
		args = append(args, sanitizer)
	}
	var identifier output.ExternalReference
	if animationKind == ir.AnimationKindEnter {
		identifier = *r3_identifiers.AnimationEnter
	} else {
		identifier = *r3_identifiers.AnimationLeave
	}
	return call(identifier, args, sourceSpan)
}

// AnimationString creates an animation string operation
func AnimationString(
	animationKind ir.AnimationKind,
	expression interface{}, // output.OutputExpression | *ops.Interpolation
	sanitizer output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	var value output.OutputExpression
	if interp, ok := expression.(*ops_update.Interpolation); ok {
		value = interpolationToExpression(interp, sourceSpan)
	} else if e, ok := expression.(output.OutputExpression); ok {
		value = e
	} else {
		panic(fmt.Sprintf("invalid expression type: %T", expression))
	}
	args := []output.OutputExpression{value}
	if sanitizer != nil {
		args = append(args, sanitizer)
	}
	var identifier output.ExternalReference
	if animationKind == ir.AnimationKindEnter {
		identifier = *r3_identifiers.AnimationEnter
	} else {
		identifier = *r3_identifiers.AnimationLeave
	}
	return call(identifier, args, sourceSpan)
}

// AnimationListener creates an animation listener operation
func AnimationListener(
	animationKind ir.AnimationKind,
	handlerFn output.OutputExpression,
	eventTargetResolver *output.ExternalReference,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.CreateOp {
	args := []output.OutputExpression{handlerFn}
	if eventTargetResolver != nil {
		args = append(args, output.NewExternalExpr(eventTargetResolver, nil, nil, nil))
	}
	var identifier output.ExternalReference
	if animationKind == ir.AnimationKindEnter {
		identifier = *r3_identifiers.AnimationEnterListener
	} else {
		identifier = *r3_identifiers.AnimationLeaveListener
	}
	return call(identifier, args, sourceSpan)
}

// SyntheticHostProperty creates a synthetic host property operation
func SyntheticHostProperty(
	name string,
	expression output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp {
	return call(
		*r3_identifiers.SyntheticHostProperty,
		[]output.OutputExpression{
			output.NewLiteralExpr(name, nil, nil),
			expression,
		},
		sourceSpan,
	)
}
