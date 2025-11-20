// Package partial provides functions for compiling Angular declarations into partial compilation format.
//
// This package contains functions for compiling various Angular entities (directives, components,
// pipes, injectables, injectors, NgModules, factories, and class metadata) into the partial
// declaration format used by the Angular linker.
//
// The partial compilation format allows libraries to be compiled separately and linked together
// at runtime, enabling features like lazy loading and code splitting.
//
// All functions in this package maintain 1:1 logic equivalence with the TypeScript implementation
// in @angular/compiler/src/render3/partial.
//
// Main exports (matching compiler.ts exports):
//
//   - CompileDeclareClassMetadata, CompileComponentDeclareClassMetadata - Class metadata compilation
//   - CompileDeclareComponentFromMetadata, DeclareComponentTemplateInfo - Component compilation
//   - CompileDeclareDirectiveFromMetadata - Directive compilation
//   - CompileDeclareFactoryFunction - Factory compilation
//   - CompileDeclareInjectableFromMetadata - Injectable compilation
//   - CompileDeclareInjectorFromMetadata - Injector compilation
//   - CompileDeclareNgModuleFromMetadata - NgModule compilation
//   - CompileDeclarePipeFromMetadata - Pipe compilation
//
// Types (from api.go):
//
//   - R3DeclareDirectiveMetadata, R3DeclareComponentMetadata - Directive/Component metadata
//   - R3DeclarePipeMetadata - Pipe metadata
//   - R3DeclareInjectableMetadata - Injectable metadata
//   - R3DeclareInjectorMetadata - Injector metadata
//   - R3DeclareNgModuleMetadata - NgModule metadata
//   - R3DeclareFactoryMetadata - Factory metadata
//   - R3DeclareClassMetadata, R3DeclareClassMetadataAsync - Class metadata
//   - R3DeclareQueryMetadata - Query metadata
//   - R3DeclareDependencyMetadata - Dependency metadata
//   - R3DeclareTemplateDependencyMetadata - Template dependency metadata
//
// Utility functions:
//
//   - CompileDependencies, CompileDependency - Dependency compilation
//   - ToOptionalLiteralArray, ToOptionalLiteralMap - Array/Map literal helpers
package partial
