// Package compiler provides the Angular compiler APIs for parsing templates, generating code,
// and compiling Angular applications.
//
// This package is a Go port of @angular/compiler, maintaining 1:1 logic equivalence with the
// TypeScript implementation.
//
// <div class="callout is-critical">
//   <header>Unstable APIs</header>
//   <p>
//     All compiler apis are currently considered experimental and private!
//   </p>
//   <p>
//     We expect the APIs in this package to keep on changing. Do not rely on them.
//   </p>
// </div>
//
// Main sub-packages:
//
//   - core: Core Angular types (ChangeDetectionStrategy, ViewEncapsulation, etc.)
//   - output: Output AST types and expressions for code generation
//   - render3: Render3 (Ivy) compilation logic
//     - partial: Partial compilation functions for directives, components, pipes, etc.
//     - view: View compilation and template processing
//     - r3_identifiers: Ivy instruction identifiers
//   - ml_parser: HTML/XML template parsing
//   - expression_parser: Expression parsing for bindings and events
//   - i18n: Internationalization support
//   - schema: Element schema registry for validation
//
// Partial Compilation Exports (from render3/partial):
//
//   Class Metadata:
//     - CompileDeclareClassMetadata
//     - CompileComponentDeclareClassMetadata
//
//   Component:
//     - CompileDeclareComponentFromMetadata
//     - DeclareComponentTemplateInfo
//
//   Directive:
//     - CompileDeclareDirectiveFromMetadata
//
//   Factory:
//     - CompileDeclareFactoryFunction
//
//   Injectable:
//     - CompileDeclareInjectableFromMetadata
//
//   Injector:
//     - CompileDeclareInjectorFromMetadata
//
//   NgModule:
//     - CompileDeclareNgModuleFromMetadata
//
//   Pipe:
//     - CompileDeclarePipeFromMetadata
//
//   Types (from render3/partial/api):
//     - R3DeclareDirectiveMetadata, R3DeclareComponentMetadata
//     - R3DeclarePipeMetadata
//     - R3DeclareInjectableMetadata
//     - R3DeclareInjectorMetadata
//     - R3DeclareNgModuleMetadata
//     - R3DeclareFactoryMetadata
//     - R3DeclareClassMetadata, R3DeclareClassMetadataAsync
//     - R3DeclareQueryMetadata
//     - R3DeclareDependencyMetadata
//     - R3DeclareTemplateDependencyMetadata
//
// View Compilation Exports (from render3/view):
//
//   - CompileComponentFromMetadata
//   - CompileDirectiveFromMetadata
//   - CompileDeferResolverFunction
//   - ParseTemplate, ParsedTemplate
//   - MakeBindingParser
//
// Render3 Exports (from render3):
//
//   - R3CompiledExpression, R3Reference
//   - MaybeForwardRefExpression
//   - CompileFactoryFunction
//   - R3DependencyMetadata
//   - R3Identifiers
//
// Template AST (from render3/r3_ast):
//
//   - Node, Element, Template, Text, etc.
//   - RecursiveVisitor, Visitor
//   - VisitAll
//
// Output AST (from output):
//
//   - OutputExpression, Expression, Statement
//   - LiteralExpr, LiteralArrayExpr, LiteralMapExpr
//   - InvokeFunctionExpr, ExternalExpr
//   - Type, ExpressionType, etc.
//
// Core Types (from core):
//
//   - ChangeDetectionStrategy
//   - ViewEncapsulation
//   - SecurityContext
//
// This file only documents the main exports. For detailed API documentation, see the individual
// package documentation.
package compiler

