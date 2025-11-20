package facade

// CompilerFacadeInterface defines interfaces shared between @angular/core and @angular/compiler
// to allow for late binding of @angular/compiler for JIT purposes.
//
// This file mirrors:
//  - packages/compiler/src/compiler_facade_interface.ts          (main)
//  - packages/core/src/compiler/compiler_facade_interface.ts   (replica)

// ExportedCompilerFacade is the exported compiler facade interface
type ExportedCompilerFacade interface {
	GetCompilerFacade() CompilerFacade
}

// CompilerFacade is the main compiler facade interface
type CompilerFacade interface {
	CompilePipe(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		meta R3PipeMetadataFacade,
	) interface{}

	CompilePipeDeclaration(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		declaration R3DeclarePipeFacade,
	) interface{}

	CompileInjectable(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		meta R3InjectableMetadataFacade,
	) interface{}

	CompileInjectableDeclaration(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		meta R3DeclareInjectableFacade,
	) interface{}

	CompileInjector(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		meta R3InjectorMetadataFacade,
	) interface{}

	CompileInjectorDeclaration(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		declaration R3DeclareInjectorFacade,
	) interface{}

	CompileNgModule(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		meta R3NgModuleMetadataFacade,
	) interface{}

	CompileNgModuleDeclaration(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		declaration R3DeclareNgModuleFacade,
	) interface{}

	CompileDirective(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		meta R3DirectiveMetadataFacade,
	) interface{}

	CompileDirectiveDeclaration(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		declaration R3DeclareDirectiveFacade,
	) interface{}

	CompileComponent(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		meta R3ComponentMetadataFacade,
	) interface{}

	CompileComponentDeclaration(
		angularCoreEnv CoreEnvironment,
		sourceMapUrl string,
		declaration R3DeclareComponentFacade,
	) interface{}
}

// CoreEnvironment represents the Angular core environment
type CoreEnvironment interface{}

// R3PipeMetadataFacade represents R3 pipe metadata facade
type R3PipeMetadataFacade interface{}

// R3DeclarePipeFacade represents R3 declare pipe facade
type R3DeclarePipeFacade interface{}

// R3InjectableMetadataFacade represents R3 injectable metadata facade
type R3InjectableMetadataFacade interface{}

// R3DeclareInjectableFacade represents R3 declare injectable facade
type R3DeclareInjectableFacade interface{}

// R3InjectorMetadataFacade represents R3 injector metadata facade
type R3InjectorMetadataFacade interface{}

// R3DeclareInjectorFacade represents R3 declare injector facade
type R3DeclareInjectorFacade interface{}

// R3NgModuleMetadataFacade represents R3 NgModule metadata facade
type R3NgModuleMetadataFacade interface{}

// R3DeclareNgModuleFacade represents R3 declare NgModule facade
type R3DeclareNgModuleFacade interface{}

// R3DirectiveMetadataFacade represents R3 directive metadata facade
type R3DirectiveMetadataFacade interface{}

// R3DeclareDirectiveFacade represents R3 declare directive facade
type R3DeclareDirectiveFacade interface{}

// R3ComponentMetadataFacade represents R3 component metadata facade
type R3ComponentMetadataFacade interface{}

// R3DeclareComponentFacade represents R3 declare component facade
type R3DeclareComponentFacade interface{}

// FactoryTarget represents the type of target being created by a factory
type FactoryTarget int

const (
	FactoryTargetDirective FactoryTarget = iota
	FactoryTargetComponent
	FactoryTargetInjectable
	FactoryTargetPipe
	FactoryTargetNgModule
)

