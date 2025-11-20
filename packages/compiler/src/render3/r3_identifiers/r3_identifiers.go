package r3_identifiers

import (
	"ngc-go/packages/compiler/src/output"
)

var CORE string = "@angular/core"

// Identifiers contains all R3 identifiers used in the compiler
type Identifiers struct{}

// Methods
const (
	NEW_METHOD       = "factory"
	TRANSFORM_METHOD = "transform"
	PATCH_DEPS       = "patchedDeps"
)

var Core = &output.ExternalReference{Name: nil, ModuleName: &CORE}

// Instructions
var (
	NamespaceHTML   = &output.ExternalReference{Name: stringPtr("ɵɵnamespaceHTML"), ModuleName: &CORE}
	NamespaceMathML = &output.ExternalReference{Name: stringPtr("ɵɵnamespaceMathML"), ModuleName: &CORE}
	NamespaceSVG    = &output.ExternalReference{Name: stringPtr("ɵɵnamespaceSVG"), ModuleName: &CORE}

	Element      = &output.ExternalReference{Name: stringPtr("ɵɵelement"), ModuleName: &CORE}
	ElementStart = &output.ExternalReference{Name: stringPtr("ɵɵelementStart"), ModuleName: &CORE}
	ElementEnd   = &output.ExternalReference{Name: stringPtr("ɵɵelementEnd"), ModuleName: &CORE}

	DomElement               = &output.ExternalReference{Name: stringPtr("ɵɵdomElement"), ModuleName: &CORE}
	DomElementStart          = &output.ExternalReference{Name: stringPtr("ɵɵdomElementStart"), ModuleName: &CORE}
	DomElementEnd            = &output.ExternalReference{Name: stringPtr("ɵɵdomElementEnd"), ModuleName: &CORE}
	DomElementContainer      = &output.ExternalReference{Name: stringPtr("ɵɵdomElementContainer"), ModuleName: &CORE}
	DomElementContainerStart = &output.ExternalReference{Name: stringPtr("ɵɵdomElementContainerStart"), ModuleName: &CORE}
	DomElementContainerEnd   = &output.ExternalReference{Name: stringPtr("ɵɵdomElementContainerEnd"), ModuleName: &CORE}
	DomTemplate              = &output.ExternalReference{Name: stringPtr("ɵɵdomTemplate"), ModuleName: &CORE}
	DomListener              = &output.ExternalReference{Name: stringPtr("ɵɵdomListener"), ModuleName: &CORE}

	Advance = &output.ExternalReference{Name: stringPtr("ɵɵadvance"), ModuleName: &CORE}

	SyntheticHostProperty = &output.ExternalReference{Name: stringPtr("ɵɵsyntheticHostProperty"), ModuleName: &CORE}
	SyntheticHostListener = &output.ExternalReference{Name: stringPtr("ɵɵsyntheticHostListener"), ModuleName: &CORE}

	Attribute = &output.ExternalReference{Name: stringPtr("ɵɵattribute"), ModuleName: &CORE}

	ClassProp = &output.ExternalReference{Name: stringPtr("ɵɵclassProp"), ModuleName: &CORE}

	ElementContainerStart = &output.ExternalReference{Name: stringPtr("ɵɵelementContainerStart"), ModuleName: &CORE}
	ElementContainerEnd   = &output.ExternalReference{Name: stringPtr("ɵɵelementContainerEnd"), ModuleName: &CORE}
	ElementContainer      = &output.ExternalReference{Name: stringPtr("ɵɵelementContainer"), ModuleName: &CORE}

	StyleMap  = &output.ExternalReference{Name: stringPtr("ɵɵstyleMap"), ModuleName: &CORE}
	ClassMap  = &output.ExternalReference{Name: stringPtr("ɵɵclassMap"), ModuleName: &CORE}
	StyleProp = &output.ExternalReference{Name: stringPtr("ɵɵstyleProp"), ModuleName: &CORE}

	Interpolate  = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate"), ModuleName: &CORE}
	Interpolate1 = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate1"), ModuleName: &CORE}
	Interpolate2 = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate2"), ModuleName: &CORE}
	Interpolate3 = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate3"), ModuleName: &CORE}
	Interpolate4 = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate4"), ModuleName: &CORE}
	Interpolate5 = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate5"), ModuleName: &CORE}
	Interpolate6 = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate6"), ModuleName: &CORE}
	Interpolate7 = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate7"), ModuleName: &CORE}
	Interpolate8 = &output.ExternalReference{Name: stringPtr("ɵɵinterpolate8"), ModuleName: &CORE}
	InterpolateV = &output.ExternalReference{Name: stringPtr("ɵɵinterpolateV"), ModuleName: &CORE}

	NextContext = &output.ExternalReference{Name: stringPtr("ɵɵnextContext"), ModuleName: &CORE}
	ResetView   = &output.ExternalReference{Name: stringPtr("ɵɵresetView"), ModuleName: &CORE}

	TemplateCreate = &output.ExternalReference{Name: stringPtr("ɵɵtemplate"), ModuleName: &CORE}

	Defer                      = &output.ExternalReference{Name: stringPtr("ɵɵdefer"), ModuleName: &CORE}
	DeferWhen                  = &output.ExternalReference{Name: stringPtr("ɵɵdeferWhen"), ModuleName: &CORE}
	DeferOnIdle                = &output.ExternalReference{Name: stringPtr("ɵɵdeferOnIdle"), ModuleName: &CORE}
	DeferOnImmediate           = &output.ExternalReference{Name: stringPtr("ɵɵdeferOnImmediate"), ModuleName: &CORE}
	DeferOnTimer               = &output.ExternalReference{Name: stringPtr("ɵɵdeferOnTimer"), ModuleName: &CORE}
	DeferOnHover               = &output.ExternalReference{Name: stringPtr("ɵɵdeferOnHover"), ModuleName: &CORE}
	DeferOnInteraction         = &output.ExternalReference{Name: stringPtr("ɵɵdeferOnInteraction"), ModuleName: &CORE}
	DeferOnViewport            = &output.ExternalReference{Name: stringPtr("ɵɵdeferOnViewport"), ModuleName: &CORE}
	DeferPrefetchWhen          = &output.ExternalReference{Name: stringPtr("ɵɵdeferPrefetchWhen"), ModuleName: &CORE}
	DeferPrefetchOnIdle        = &output.ExternalReference{Name: stringPtr("ɵɵdeferPrefetchOnIdle"), ModuleName: &CORE}
	DeferPrefetchOnImmediate   = &output.ExternalReference{Name: stringPtr("ɵɵdeferPrefetchOnImmediate"), ModuleName: &CORE}
	DeferPrefetchOnTimer       = &output.ExternalReference{Name: stringPtr("ɵɵdeferPrefetchOnTimer"), ModuleName: &CORE}
	DeferPrefetchOnHover       = &output.ExternalReference{Name: stringPtr("ɵɵdeferPrefetchOnHover"), ModuleName: &CORE}
	DeferPrefetchOnInteraction = &output.ExternalReference{Name: stringPtr("ɵɵdeferPrefetchOnInteraction"), ModuleName: &CORE}
	DeferPrefetchOnViewport    = &output.ExternalReference{Name: stringPtr("ɵɵdeferPrefetchOnViewport"), ModuleName: &CORE}
	DeferHydrateWhen           = &output.ExternalReference{Name: stringPtr("ɵɵdeferHydrateWhen"), ModuleName: &CORE}
	DeferHydrateNever          = &output.ExternalReference{Name: stringPtr("ɵɵdeferHydrateNever"), ModuleName: &CORE}
	DeferHydrateOnIdle         = &output.ExternalReference{Name: stringPtr("ɵɵdeferHydrateOnIdle"), ModuleName: &CORE}
	DeferHydrateOnImmediate    = &output.ExternalReference{Name: stringPtr("ɵɵdeferHydrateOnImmediate"), ModuleName: &CORE}
	DeferHydrateOnTimer        = &output.ExternalReference{Name: stringPtr("ɵɵdeferHydrateOnTimer"), ModuleName: &CORE}
	DeferHydrateOnHover        = &output.ExternalReference{Name: stringPtr("ɵɵdeferHydrateOnHover"), ModuleName: &CORE}
	DeferHydrateOnInteraction  = &output.ExternalReference{Name: stringPtr("ɵɵdeferHydrateOnInteraction"), ModuleName: &CORE}
	DeferHydrateOnViewport     = &output.ExternalReference{Name: stringPtr("ɵɵdeferHydrateOnViewport"), ModuleName: &CORE}
	DeferEnableTimerScheduling = &output.ExternalReference{Name: stringPtr("ɵɵdeferEnableTimerScheduling"), ModuleName: &CORE}

	ConditionalCreate       = &output.ExternalReference{Name: stringPtr("ɵɵconditionalCreate"), ModuleName: &CORE}
	ConditionalBranchCreate = &output.ExternalReference{Name: stringPtr("ɵɵconditionalBranchCreate"), ModuleName: &CORE}
	Conditional             = &output.ExternalReference{Name: stringPtr("ɵɵconditional"), ModuleName: &CORE}
	Repeater                = &output.ExternalReference{Name: stringPtr("ɵɵrepeater"), ModuleName: &CORE}
	RepeaterCreate          = &output.ExternalReference{Name: stringPtr("ɵɵrepeaterCreate"), ModuleName: &CORE}
	RepeaterTrackByIndex    = &output.ExternalReference{Name: stringPtr("ɵɵrepeaterTrackByIndex"), ModuleName: &CORE}
	RepeaterTrackByIdentity = &output.ExternalReference{Name: stringPtr("ɵɵrepeaterTrackByIdentity"), ModuleName: &CORE}
	ComponentInstance       = &output.ExternalReference{Name: stringPtr("ɵɵcomponentInstance"), ModuleName: &CORE}

	Text = &output.ExternalReference{Name: stringPtr("ɵɵtext"), ModuleName: &CORE}

	EnableBindings  = &output.ExternalReference{Name: stringPtr("ɵɵenableBindings"), ModuleName: &CORE}
	DisableBindings = &output.ExternalReference{Name: stringPtr("ɵɵdisableBindings"), ModuleName: &CORE}

	GetCurrentView = &output.ExternalReference{Name: stringPtr("ɵɵgetCurrentView"), ModuleName: &CORE}

	TextInterpolate  = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate"), ModuleName: &CORE}
	TextInterpolate1 = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate1"), ModuleName: &CORE}
	TextInterpolate2 = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate2"), ModuleName: &CORE}
	TextInterpolate3 = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate3"), ModuleName: &CORE}
	TextInterpolate4 = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate4"), ModuleName: &CORE}
	TextInterpolate5 = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate5"), ModuleName: &CORE}
	TextInterpolate6 = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate6"), ModuleName: &CORE}
	TextInterpolate7 = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate7"), ModuleName: &CORE}
	TextInterpolate8 = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolate8"), ModuleName: &CORE}
	TextInterpolateV = &output.ExternalReference{Name: stringPtr("ɵɵtextInterpolateV"), ModuleName: &CORE}

	RestoreView = &output.ExternalReference{Name: stringPtr("ɵɵrestoreView"), ModuleName: &CORE}

	PureFunction0 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction0"), ModuleName: &CORE}
	PureFunction1 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction1"), ModuleName: &CORE}
	PureFunction2 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction2"), ModuleName: &CORE}
	PureFunction3 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction3"), ModuleName: &CORE}
	PureFunction4 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction4"), ModuleName: &CORE}
	PureFunction5 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction5"), ModuleName: &CORE}
	PureFunction6 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction6"), ModuleName: &CORE}
	PureFunction7 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction7"), ModuleName: &CORE}
	PureFunction8 = &output.ExternalReference{Name: stringPtr("ɵɵpureFunction8"), ModuleName: &CORE}
	PureFunctionV = &output.ExternalReference{Name: stringPtr("ɵɵpureFunctionV"), ModuleName: &CORE}

	PipeBind1 = &output.ExternalReference{Name: stringPtr("ɵɵpipeBind1"), ModuleName: &CORE}
	PipeBind2 = &output.ExternalReference{Name: stringPtr("ɵɵpipeBind2"), ModuleName: &CORE}
	PipeBind3 = &output.ExternalReference{Name: stringPtr("ɵɵpipeBind3"), ModuleName: &CORE}
	PipeBind4 = &output.ExternalReference{Name: stringPtr("ɵɵpipeBind4"), ModuleName: &CORE}
	PipeBindV = &output.ExternalReference{Name: stringPtr("ɵɵpipeBindV"), ModuleName: &CORE}

	DomProperty  = &output.ExternalReference{Name: stringPtr("ɵɵdomProperty"), ModuleName: &CORE}
	AriaProperty = &output.ExternalReference{Name: stringPtr("ɵɵariaProperty"), ModuleName: &CORE}
	Property     = &output.ExternalReference{Name: stringPtr("ɵɵproperty"), ModuleName: &CORE}

	Control       = &output.ExternalReference{Name: stringPtr("ɵɵcontrol"), ModuleName: &CORE}
	ControlCreate = &output.ExternalReference{Name: stringPtr("ɵɵcontrolCreate"), ModuleName: &CORE}

	AnimationEnterListener = &output.ExternalReference{Name: stringPtr("ɵɵanimateEnterListener"), ModuleName: &CORE}
	AnimationLeaveListener = &output.ExternalReference{Name: stringPtr("ɵɵanimateLeaveListener"), ModuleName: &CORE}
	AnimationEnter         = &output.ExternalReference{Name: stringPtr("ɵɵanimateEnter"), ModuleName: &CORE}
	AnimationLeave         = &output.ExternalReference{Name: stringPtr("ɵɵanimateLeave"), ModuleName: &CORE}

	I18n            = &output.ExternalReference{Name: stringPtr("ɵɵi18n"), ModuleName: &CORE}
	I18nAttributes  = &output.ExternalReference{Name: stringPtr("ɵɵi18nAttributes"), ModuleName: &CORE}
	I18nExp         = &output.ExternalReference{Name: stringPtr("ɵɵi18nExp"), ModuleName: &CORE}
	I18nStart       = &output.ExternalReference{Name: stringPtr("ɵɵi18nStart"), ModuleName: &CORE}
	I18nEnd         = &output.ExternalReference{Name: stringPtr("ɵɵi18nEnd"), ModuleName: &CORE}
	I18nApply       = &output.ExternalReference{Name: stringPtr("ɵɵi18nApply"), ModuleName: &CORE}
	I18nPostprocess = &output.ExternalReference{Name: stringPtr("ɵɵi18nPostprocess"), ModuleName: &CORE}

	Pipe = &output.ExternalReference{Name: stringPtr("ɵɵpipe"), ModuleName: &CORE}

	Projection    = &output.ExternalReference{Name: stringPtr("ɵɵprojection"), ModuleName: &CORE}
	ProjectionDef = &output.ExternalReference{Name: stringPtr("ɵɵprojectionDef"), ModuleName: &CORE}

	Reference = &output.ExternalReference{Name: stringPtr("ɵɵreference"), ModuleName: &CORE}

	Inject = &output.ExternalReference{Name: stringPtr("ɵɵinject"), ModuleName: &CORE}

	InjectAttribute = &output.ExternalReference{Name: stringPtr("ɵɵinjectAttribute"), ModuleName: &CORE}

	DirectiveInject   = &output.ExternalReference{Name: stringPtr("ɵɵdirectiveInject"), ModuleName: &CORE}
	InvalidFactory    = &output.ExternalReference{Name: stringPtr("ɵɵinvalidFactory"), ModuleName: &CORE}
	InvalidFactoryDep = &output.ExternalReference{Name: stringPtr("ɵɵinvalidFactoryDep"), ModuleName: &CORE}

	TemplateRefExtractor = &output.ExternalReference{Name: stringPtr("ɵɵtemplateRefExtractor"), ModuleName: &CORE}

	ForwardRef        = &output.ExternalReference{Name: stringPtr("forwardRef"), ModuleName: &CORE}
	ResolveForwardRef = &output.ExternalReference{Name: stringPtr("resolveForwardRef"), ModuleName: &CORE}

	ReplaceMetadata       = &output.ExternalReference{Name: stringPtr("ɵɵreplaceMetadata"), ModuleName: &CORE}
	GetReplaceMetadataURL = &output.ExternalReference{Name: stringPtr("ɵɵgetReplaceMetadataURL"), ModuleName: &CORE}

	DefineInjectable      = &output.ExternalReference{Name: stringPtr("ɵɵdefineInjectable"), ModuleName: &CORE}
	DeclareInjectable     = &output.ExternalReference{Name: stringPtr("ɵɵngDeclareInjectable"), ModuleName: &CORE}
	InjectableDeclaration = &output.ExternalReference{Name: stringPtr("ɵɵInjectableDeclaration"), ModuleName: &CORE}

	ResolveWindow   = &output.ExternalReference{Name: stringPtr("ɵɵresolveWindow"), ModuleName: &CORE}
	ResolveDocument = &output.ExternalReference{Name: stringPtr("ɵɵresolveDocument"), ModuleName: &CORE}
	ResolveBody     = &output.ExternalReference{Name: stringPtr("ɵɵresolveBody"), ModuleName: &CORE}

	GetComponentDepsFactory = &output.ExternalReference{Name: stringPtr("ɵɵgetComponentDepsFactory"), ModuleName: &CORE}

	DefineComponent  = &output.ExternalReference{Name: stringPtr("ɵɵdefineComponent"), ModuleName: &CORE}
	DeclareComponent = &output.ExternalReference{Name: stringPtr("ɵɵngDeclareComponent"), ModuleName: &CORE}

	SetComponentScope = &output.ExternalReference{Name: stringPtr("ɵɵsetComponentScope"), ModuleName: &CORE}

	ChangeDetectionStrategy = &output.ExternalReference{Name: stringPtr("ChangeDetectionStrategy"), ModuleName: &CORE}
	ViewEncapsulation       = &output.ExternalReference{Name: stringPtr("ViewEncapsulation"), ModuleName: &CORE}

	ComponentDeclaration = &output.ExternalReference{Name: stringPtr("ɵɵComponentDeclaration"), ModuleName: &CORE}

	FactoryDeclaration = &output.ExternalReference{Name: stringPtr("ɵɵFactoryDeclaration"), ModuleName: &CORE}
	DeclareFactory     = &output.ExternalReference{Name: stringPtr("ɵɵngDeclareFactory"), ModuleName: &CORE}
	FactoryTarget      = &output.ExternalReference{Name: stringPtr("ɵɵFactoryTarget"), ModuleName: &CORE}

	DefineDirective  = &output.ExternalReference{Name: stringPtr("ɵɵdefineDirective"), ModuleName: &CORE}
	DeclareDirective = &output.ExternalReference{Name: stringPtr("ɵɵngDeclareDirective"), ModuleName: &CORE}

	DirectiveDeclaration = &output.ExternalReference{Name: stringPtr("ɵɵDirectiveDeclaration"), ModuleName: &CORE}

	InjectorDef         = &output.ExternalReference{Name: stringPtr("ɵɵInjectorDef"), ModuleName: &CORE}
	InjectorDeclaration = &output.ExternalReference{Name: stringPtr("ɵɵInjectorDeclaration"), ModuleName: &CORE}

	DefineInjector  = &output.ExternalReference{Name: stringPtr("ɵɵdefineInjector"), ModuleName: &CORE}
	DeclareInjector = &output.ExternalReference{Name: stringPtr("ɵɵngDeclareInjector"), ModuleName: &CORE}

	NgModuleDeclaration = &output.ExternalReference{Name: stringPtr("ɵɵNgModuleDeclaration"), ModuleName: &CORE}

	ModuleWithProviders = &output.ExternalReference{Name: stringPtr("ModuleWithProviders"), ModuleName: &CORE}

	DefineNgModule       = &output.ExternalReference{Name: stringPtr("ɵɵdefineNgModule"), ModuleName: &CORE}
	DeclareNgModule      = &output.ExternalReference{Name: stringPtr("ɵɵngDeclareNgModule"), ModuleName: &CORE}
	SetNgModuleScope     = &output.ExternalReference{Name: stringPtr("ɵɵsetNgModuleScope"), ModuleName: &CORE}
	RegisterNgModuleType = &output.ExternalReference{Name: stringPtr("ɵɵregisterNgModuleType"), ModuleName: &CORE}

	PipeDeclaration = &output.ExternalReference{Name: stringPtr("ɵɵPipeDeclaration"), ModuleName: &CORE}

	DefinePipe  = &output.ExternalReference{Name: stringPtr("ɵɵdefinePipe"), ModuleName: &CORE}
	DeclarePipe = &output.ExternalReference{Name: stringPtr("ɵɵngDeclarePipe"), ModuleName: &CORE}

	DeclareClassMetadata      = &output.ExternalReference{Name: stringPtr("ɵɵngDeclareClassMetadata"), ModuleName: &CORE}
	DeclareClassMetadataAsync = &output.ExternalReference{Name: stringPtr("ɵɵngDeclareClassMetadataAsync"), ModuleName: &CORE}
	SetClassMetadata          = &output.ExternalReference{Name: stringPtr("ɵsetClassMetadata"), ModuleName: &CORE}
	SetClassMetadataAsync     = &output.ExternalReference{Name: stringPtr("ɵsetClassMetadataAsync"), ModuleName: &CORE}
	SetClassDebugInfo         = &output.ExternalReference{Name: stringPtr("ɵsetClassDebugInfo"), ModuleName: &CORE}
	QueryRefresh              = &output.ExternalReference{Name: stringPtr("ɵɵqueryRefresh"), ModuleName: &CORE}
	ViewQuery                 = &output.ExternalReference{Name: stringPtr("ɵɵviewQuery"), ModuleName: &CORE}
	LoadQuery                 = &output.ExternalReference{Name: stringPtr("ɵɵloadQuery"), ModuleName: &CORE}
	ContentQuery              = &output.ExternalReference{Name: stringPtr("ɵɵcontentQuery"), ModuleName: &CORE}

	// Signal queries
	ViewQuerySignal    = &output.ExternalReference{Name: stringPtr("ɵɵviewQuerySignal"), ModuleName: &CORE}
	ContentQuerySignal = &output.ExternalReference{Name: stringPtr("ɵɵcontentQuerySignal"), ModuleName: &CORE}
	QueryAdvance       = &output.ExternalReference{Name: stringPtr("ɵɵqueryAdvance"), ModuleName: &CORE}

	// Two-way bindings
	TwoWayProperty   = &output.ExternalReference{Name: stringPtr("ɵɵtwoWayProperty"), ModuleName: &CORE}
	TwoWayBindingSet = &output.ExternalReference{Name: stringPtr("ɵɵtwoWayBindingSet"), ModuleName: &CORE}
	TwoWayListener   = &output.ExternalReference{Name: stringPtr("ɵɵtwoWayListener"), ModuleName: &CORE}

	DeclareLet     = &output.ExternalReference{Name: stringPtr("ɵɵdeclareLet"), ModuleName: &CORE}
	StoreLet       = &output.ExternalReference{Name: stringPtr("ɵɵstoreLet"), ModuleName: &CORE}
	ReadContextLet = &output.ExternalReference{Name: stringPtr("ɵɵreadContextLet"), ModuleName: &CORE}

	AttachSourceLocations = &output.ExternalReference{Name: stringPtr("ɵɵattachSourceLocations"), ModuleName: &CORE}

	NgOnChangesFeature = &output.ExternalReference{Name: stringPtr("ɵɵNgOnChangesFeature"), ModuleName: &CORE}

	InheritDefinitionFeature = &output.ExternalReference{Name: stringPtr("ɵɵInheritDefinitionFeature"), ModuleName: &CORE}

	ProvidersFeature = &output.ExternalReference{Name: stringPtr("ɵɵProvidersFeature"), ModuleName: &CORE}

	HostDirectivesFeature = &output.ExternalReference{Name: stringPtr("ɵɵHostDirectivesFeature"), ModuleName: &CORE}

	ExternalStylesFeature = &output.ExternalReference{Name: stringPtr("ɵɵExternalStylesFeature"), ModuleName: &CORE}

	Listener = &output.ExternalReference{Name: stringPtr("ɵɵlistener"), ModuleName: &CORE}

	GetInheritedFactory = &output.ExternalReference{Name: stringPtr("ɵɵgetInheritedFactory"), ModuleName: &CORE}

	// sanitization-related functions
	SanitizeHtml             = &output.ExternalReference{Name: stringPtr("ɵɵsanitizeHtml"), ModuleName: &CORE}
	SanitizeStyle            = &output.ExternalReference{Name: stringPtr("ɵɵsanitizeStyle"), ModuleName: &CORE}
	SanitizeResourceUrl      = &output.ExternalReference{Name: stringPtr("ɵɵsanitizeResourceUrl"), ModuleName: &CORE}
	SanitizeScript           = &output.ExternalReference{Name: stringPtr("ɵɵsanitizeScript"), ModuleName: &CORE}
	SanitizeUrl              = &output.ExternalReference{Name: stringPtr("ɵɵsanitizeUrl"), ModuleName: &CORE}
	SanitizeUrlOrResourceUrl = &output.ExternalReference{Name: stringPtr("ɵɵsanitizeUrlOrResourceUrl"), ModuleName: &CORE}
	TrustConstantHtml        = &output.ExternalReference{Name: stringPtr("ɵɵtrustConstantHtml"), ModuleName: &CORE}
	TrustConstantResourceUrl = &output.ExternalReference{Name: stringPtr("ɵɵtrustConstantResourceUrl"), ModuleName: &CORE}
	ValidateIframeAttribute  = &output.ExternalReference{Name: stringPtr("ɵɵvalidateIframeAttribute"), ModuleName: &CORE}

	// Decorators
	InputDecorator           = &output.ExternalReference{Name: stringPtr("Input"), ModuleName: &CORE}
	OutputDecorator          = &output.ExternalReference{Name: stringPtr("Output"), ModuleName: &CORE}
	ViewChildDecorator       = &output.ExternalReference{Name: stringPtr("ViewChild"), ModuleName: &CORE}
	ViewChildrenDecorator    = &output.ExternalReference{Name: stringPtr("ViewChildren"), ModuleName: &CORE}
	ContentChildDecorator    = &output.ExternalReference{Name: stringPtr("ContentChild"), ModuleName: &CORE}
	ContentChildrenDecorator = &output.ExternalReference{Name: stringPtr("ContentChildren"), ModuleName: &CORE}

	// type-checking
	InputSignalBrandWriteType   = &output.ExternalReference{Name: stringPtr("ɵINPUT_SIGNAL_BRAND_WRITE_TYPE"), ModuleName: &CORE}
	UnwrapDirectiveSignalInputs = &output.ExternalReference{Name: stringPtr("ɵUnwrapDirectiveSignalInputs"), ModuleName: &CORE}
	UnwrapWritableSignal        = &output.ExternalReference{Name: stringPtr("ɵunwrapWritableSignal"), ModuleName: &CORE}
	AssertType                  = &output.ExternalReference{Name: stringPtr("ɵassertType"), ModuleName: &CORE}
)

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}
