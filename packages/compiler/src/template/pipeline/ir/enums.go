package ir

// OpKind distinguishes different kinds of IR operations
type OpKind int

const (
	// OpKindListEnd - A special operations type which is used to represent the beginning and end nodes of a linked list of operations
	OpKindListEnd OpKind = iota
	// OpKindStatement - An operations which wraps an output AST statement
	OpKindStatement
	// OpKindVariable - An operations which declares and initializes a `SemanticVariable`
	OpKindVariable
	// OpKindElementStart - An operations to begin rendering of an element
	OpKindElementStart
	// OpKindElement - An operations to render an element with no children
	OpKindElement
	// OpKindTemplate - An operations which declares an embedded view
	OpKindTemplate
	// OpKindElementEnd - An operations to end rendering of an element previously started with `ElementStart`
	OpKindElementEnd
	// OpKindContainerStart - An operations to begin an `ng-container`
	OpKindContainerStart
	// OpKindContainer - An operations for an `ng-container` with no children
	OpKindContainer
	// OpKindContainerEnd - An operations to end an `ng-container`
	OpKindContainerEnd
	// OpKindDisableBindings - An operations disable binding for subsequent elements
	OpKindDisableBindings
	// OpKindConditionalCreate - Create a conditional creation instruction op
	OpKindConditionalCreate
	// OpKindConditionalBranchCreate - Create a conditional branch creation instruction op
	OpKindConditionalBranchCreate
	// OpKindConditional - An op to conditionally render a template
	OpKindConditional
	// OpKindEnableBindings - An operations to re-enable binding, after it was previously disabled
	OpKindEnableBindings
	// OpKindText - An operations to render a text node
	OpKindText
	// OpKindListener - An operations declaring an event listener for an element
	OpKindListener
	// OpKindInterpolateText - An operations to interpolate text into a text node
	OpKindInterpolateText
	// OpKindBinding - An intermediate binding op, that has not yet been processed
	OpKindBinding
	// OpKindProperty - An operations to bind an expression to a property of an element
	OpKindProperty
	// OpKindStyleProp - An operations to bind an expression to a style property of an element
	OpKindStyleProp
	// OpKindClassProp - An operations to bind an expression to a class property of an element
	OpKindClassProp
	// OpKindStyleMap - An operations to bind an expression to the styles of an element
	OpKindStyleMap
	// OpKindClassMap - An operations to bind an expression to the classes of an element
	OpKindClassMap
	// OpKindAdvance - An operations to advance the runtime's implicit slot context
	OpKindAdvance
	// OpKindPipe - An operations to instantiate a pipe
	OpKindPipe
	// OpKindAttribute - An operations to associate an attribute with an element
	OpKindAttribute
	// OpKindExtractedAttribute - An attribute that has been extracted for inclusion in the consts array
	OpKindExtractedAttribute
	// OpKindDefer - An operations that configures a `@defer` block
	OpKindDefer
	// OpKindDeferOn - An operations that controls when a `@defer` loads
	OpKindDeferOn
	// OpKindDeferWhen - An operations that controls when a `@defer` loads, using a custom expression
	OpKindDeferWhen
	// OpKindI18nMessage - An i18n message that has been extracted for inclusion in the consts array
	OpKindI18nMessage
	// OpKindDomProperty - A binding to a native DOM property
	OpKindDomProperty
	// OpKindNamespace - A namespace change
	OpKindNamespace
	// OpKindProjectionDef - Configure a content projection definition for the view
	OpKindProjectionDef
	// OpKindProjection - Create a content projection slot
	OpKindProjection
	// OpKindRepeaterCreate - Create a repeater creation instruction op
	OpKindRepeaterCreate
	// OpKindRepeater - An update up for a repeater
	OpKindRepeater
	// OpKindTwoWayProperty - An operations to bind an expression to the property side of a two-way binding
	OpKindTwoWayProperty
	// OpKindTwoWayListener - An operations declaring the event side of a two-way binding
	OpKindTwoWayListener
	// OpKindDeclareLet - A creation-time operations that initializes the slot for a `@let` declaration
	OpKindDeclareLet
	// OpKindStoreLet - An update-time operations that stores the current value of a `@let` declaration
	OpKindStoreLet
	// OpKindI18nStart - The start of an i18n block
	OpKindI18nStart
	// OpKindI18n - A self-closing i18n on a single element
	OpKindI18n
	// OpKindI18nEnd - The end of an i18n block
	OpKindI18nEnd
	// OpKindI18nExpression - An expression in an i18n message
	OpKindI18nExpression
	// OpKindI18nApply - An instruction that applies a set of i18n expressions
	OpKindI18nApply
	// OpKindIcuStart - An instruction to create an ICU expression
	OpKindIcuStart
	// OpKindIcuEnd - An instruction to update an ICU expression
	OpKindIcuEnd
	// OpKindIcuPlaceholder - An instruction representing a placeholder in an ICU expression
	OpKindIcuPlaceholder
	// OpKindI18nContext - An i18n context containing information needed to generate an i18n message
	OpKindI18nContext
	// OpKindI18nAttributes - A creation op that corresponds to i18n attributes on an element
	OpKindI18nAttributes
	// OpKindSourceLocation - Creation op that attaches the location at which an element was defined
	OpKindSourceLocation
	// OpKindAnimation - An operations to bind animation css classes to an element
	OpKindAnimation
	// OpKindAnimationString - An operations to bind animation css classes to an element
	OpKindAnimationString
	// OpKindAnimationBinding - An operations to bind animation css classes to an element
	OpKindAnimationBinding
	// OpKindAnimationListener - An operations to bind animation events to an element
	OpKindAnimationListener
	// OpKindControl - An operations to bind an expression to a `field` property of an element
	OpKindControl
	// OpKindControlCreate - An operations to set up a corresponding Control operations
	OpKindControlCreate
)

// ExpressionKind distinguishes different kinds of IR expressions
type ExpressionKind int

const (
	// ExpressionKindLexicalRead - Read of a variable in a lexical scope
	ExpressionKindLexicalRead ExpressionKind = iota
	// ExpressionKindContext - A reference to the current view context
	ExpressionKindContext
	// ExpressionKindTrackContext - A reference to the view context, for use inside a track function
	ExpressionKindTrackContext
	// ExpressionKindReadVariable - Read of a variable declared in a `VariableOp`
	ExpressionKindReadVariable
	// ExpressionKindNextContext - Runtime operations to navigate to the next view context
	ExpressionKindNextContext
	// ExpressionKindReference - Runtime operations to retrieve the value of a local reference
	ExpressionKindReference
	// ExpressionKindStoreLet - A call storing the value of a `@let` declaration
	ExpressionKindStoreLet
	// ExpressionKindContextLetReference - A reference to a `@let` declaration read from the context view
	ExpressionKindContextLetReference
	// ExpressionKindGetCurrentView - Runtime operations to snapshot the current view context
	ExpressionKindGetCurrentView
	// ExpressionKindRestoreView - Runtime operations to restore a snapshotted view
	ExpressionKindRestoreView
	// ExpressionKindResetView - Runtime operations to reset the current view context after `RestoreView`
	ExpressionKindResetView
	// ExpressionKindPureFunctionExpr - Defines and calls a function with change-detected arguments
	ExpressionKindPureFunctionExpr
	// ExpressionKindPureFunctionParameterExpr - Indicates a positional parameter to a pure function definition
	ExpressionKindPureFunctionParameterExpr
	// ExpressionKindPipeBinding - Binding to a pipe transformation
	ExpressionKindPipeBinding
	// ExpressionKindPipeBindingVariadic - Binding to a pipe transformation with a variable number of arguments
	ExpressionKindPipeBindingVariadic
	// ExpressionKindSafePropertyRead - A safe property read requiring expansion into a null check
	ExpressionKindSafePropertyRead
	// ExpressionKindSafeKeyedRead - A safe keyed read requiring expansion into a null check
	ExpressionKindSafeKeyedRead
	// ExpressionKindSafeInvokeFunction - A safe function call requiring expansion into a null check
	ExpressionKindSafeInvokeFunction
	// ExpressionKindSafeTernaryExpr - An intermediate expression that will be expanded from a safe read
	ExpressionKindSafeTernaryExpr
	// ExpressionKindEmptyExpr - An empty expression that will be stripped before generating the final output
	ExpressionKindEmptyExpr
	// ExpressionKindAssignTemporaryExpr - An assignment to a temporary variable
	ExpressionKindAssignTemporaryExpr
	// ExpressionKindReadTemporaryExpr - A reference to a temporary variable
	ExpressionKindReadTemporaryExpr
	// ExpressionKindSlotLiteralExpr - An expression that will cause a literal slot index to be emitted
	ExpressionKindSlotLiteralExpr
	// ExpressionKindConditionalCase - A test expression for a conditional op
	ExpressionKindConditionalCase
	// ExpressionKindConstCollected - An expression that will be automatically extracted to the component const array
	ExpressionKindConstCollected
	// ExpressionKindTwoWayBindingSet - Operation that sets the value of a two-way binding
	ExpressionKindTwoWayBindingSet
)

// VariableFlags describes flags for variables
type VariableFlags int

const (
	// VariableFlagsNone - No flags
	VariableFlagsNone VariableFlags = 0
	// VariableFlagsAlwaysInline - Always inline this variable, regardless of the number of times it's used
	VariableFlagsAlwaysInline VariableFlags = 0b0001
)

// SemanticVariableKind distinguishes between different kinds of `SemanticVariable`s
type SemanticVariableKind int

const (
	// SemanticVariableKindContext - Represents the context of a particular view
	SemanticVariableKindContext SemanticVariableKind = iota
	// SemanticVariableKindIdentifier - Represents an identifier declared in the lexical scope of a view
	SemanticVariableKindIdentifier
	// SemanticVariableKindSavedView - Represents a saved state that can be used to restore a view
	SemanticVariableKindSavedView
	// SemanticVariableKindAlias - An alias generated by a special embedded view type
	SemanticVariableKindAlias
)

// CompatibilityMode - Whether to compile in compatibility mode
type CompatibilityMode int

const (
	// CompatibilityModeNormal - Normal compilation mode
	CompatibilityModeNormal CompatibilityMode = iota
	// CompatibilityModeTemplateDefinitionBuilder - Compatibility mode matching TemplateDefinitionBuilder output
	CompatibilityModeTemplateDefinitionBuilder
)

// BindingKind - Enumeration of the types of attributes which can be applied to an element
type BindingKind int

const (
	// BindingKindAttribute - Static attributes
	BindingKindAttribute BindingKind = iota
	// BindingKindClassName - Class bindings
	BindingKindClassName
	// BindingKindStyleProperty - Style bindings
	BindingKindStyleProperty
	// BindingKindProperty - Dynamic property bindings
	BindingKindProperty
	// BindingKindTemplate - Property or attribute bindings on a template
	BindingKindTemplate
	// BindingKindI18n - Internationalized attributes
	BindingKindI18n
	// BindingKindLegacyAnimation - Legacy animation property bindings
	BindingKindLegacyAnimation
	// BindingKindTwoWayProperty - Property side of a two-way binding
	BindingKindTwoWayProperty
	// BindingKindAnimation - Property side of an animation binding
	BindingKindAnimation
)

// I18nParamResolutionTime - Enumeration of possible times i18n params can be resolved
type I18nParamResolutionTime int

const (
	// I18nParamResolutionTimeCreation - Param is resolved at message creation time
	I18nParamResolutionTimeCreation I18nParamResolutionTime = iota
	// I18nParamResolutionTimePostprocessing - Param is resolved during post-processing
	I18nParamResolutionTimePostprocessing
)

// I18nExpressionFor - The contexts in which an i18n expression can be used
type I18nExpressionFor int

const (
	// I18nExpressionForI18nText - This expression is used as a value (i.e. inside an i18n block)
	I18nExpressionForI18nText I18nExpressionFor = iota
	// I18nExpressionForI18nAttribute - This expression is used in a binding
	I18nExpressionForI18nAttribute
)

// I18nParamValueFlags - Flags that describe what an i18n param value
type I18nParamValueFlags int

const (
	// I18nParamValueFlagsNone - No flags
	I18nParamValueFlagsNone I18nParamValueFlags = 0
	// I18nParamValueFlagsElementTag - This value represents an element tag
	I18nParamValueFlagsElementTag I18nParamValueFlags = 0b1
	// I18nParamValueFlagsTemplateTag - This value represents a template tag
	I18nParamValueFlagsTemplateTag I18nParamValueFlags = 0b10
	// I18nParamValueFlagsOpenTag - This value represents the opening of a tag
	I18nParamValueFlagsOpenTag I18nParamValueFlags = 0b0100
	// I18nParamValueFlagsCloseTag - This value represents the closing of a tag
	I18nParamValueFlagsCloseTag I18nParamValueFlags = 0b1000
	// I18nParamValueFlagsExpressionIndex - This value represents an i18n expression index
	I18nParamValueFlagsExpressionIndex I18nParamValueFlags = 0b10000
)

// Namespace - Whether the active namespace is HTML, MathML, or SVG mode
type Namespace int

const (
	// NamespaceHTML - HTML namespace
	NamespaceHTML Namespace = iota
	// NamespaceSVG - SVG namespace
	NamespaceSVG
	// NamespaceMath - MathML namespace
	NamespaceMath
)

// DeferTriggerKind - The type of a `@defer` trigger
type DeferTriggerKind int

const (
	// DeferTriggerKindIdle - Idle trigger
	DeferTriggerKindIdle DeferTriggerKind = iota
	// DeferTriggerKindImmediate - Immediate trigger
	DeferTriggerKindImmediate
	// DeferTriggerKindTimer - Timer trigger
	DeferTriggerKindTimer
	// DeferTriggerKindHover - Hover trigger
	DeferTriggerKindHover
	// DeferTriggerKindInteraction - Interaction trigger
	DeferTriggerKindInteraction
	// DeferTriggerKindViewport - Viewport trigger
	DeferTriggerKindViewport
	// DeferTriggerKindNever - Never trigger
	DeferTriggerKindNever
)

// I18nContextKind - Kinds of i18n contexts
type I18nContextKind int

const (
	// I18nContextKindRootI18n - Root i18n context
	I18nContextKindRootI18n I18nContextKind = iota
	// I18nContextKindIcu - ICU context
	I18nContextKindIcu
	// I18nContextKindAttr - Attribute context
	I18nContextKindAttr
)

// TemplateKind - Kinds of templates
type TemplateKind int

const (
	// TemplateKindNgTemplate - ng-template
	TemplateKindNgTemplate TemplateKind = iota
	// TemplateKindStructural - Structural template
	TemplateKindStructural
	// TemplateKindBlock - Block template
	TemplateKindBlock
)

// AnimationKind - Kinds of animations
type AnimationKind string

const (
	// AnimationKindEnter - Enter animation
	AnimationKindEnter AnimationKind = "enter"
	// AnimationKindLeave - Leave animation
	AnimationKindLeave AnimationKind = "leave"
)

// AnimationBindingKind - Kinds of animation bindings
type AnimationBindingKind int

const (
	// AnimationBindingKindString - String animation binding
	AnimationBindingKindString AnimationBindingKind = iota
	// AnimationBindingKindValue - Value animation binding
	AnimationBindingKindValue
)

// DeferOpModifierKind - Kinds of modifiers for a defer block
type DeferOpModifierKind string

const (
	// DeferOpModifierKindNone - No modifier
	DeferOpModifierKindNone DeferOpModifierKind = "none"
	// DeferOpModifierKindPrefetch - Prefetch modifier
	DeferOpModifierKindPrefetch DeferOpModifierKind = "prefetch"
	// DeferOpModifierKindHydrate - Hydrate modifier
	DeferOpModifierKindHydrate DeferOpModifierKind = "hydrate"
)

// TDeferDetailsFlags - Specifies defer block flags
type TDeferDetailsFlags int

const (
	// TDeferDetailsFlagsDefault - Default flags
	TDeferDetailsFlagsDefault TDeferDetailsFlags = 0
	// TDeferDetailsFlagsHasHydrateTriggers - Whether or not the defer block has hydrate triggers
	TDeferDetailsFlagsHasHydrateTriggers TDeferDetailsFlags = 1 << 0
)
