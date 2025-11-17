# Mapping: TypeScript (@angular/compiler + compiler-cli) → Go (ngc-go)

## Overview

- **Target**: Angular v18+ (latest), AOT compilation focus
- **Approach**: Hybrid (Go for template parsing + codegen; Node helper for TypeScript type info)
- **Structure**: Go code in `/Users/truong/Documents/go/packages/` (mirrors TS layout)

## @angular/compiler → Go packages

| TypeScript Path | Go Package | Purpose |
|---|---|---|
| `src/ml_parser/` | `packages/compiler/mlparser/` | Template lexer/parser |
| `src/template_parser/` | `packages/compiler/templateparser/` | Angular template AST & parser |
| `src/expression_parser/` | `packages/compiler/exprparser/` | Angular expression parser (bindings, events) |
| `src/output/` | `packages/compiler/output/` | Code generation (factory functions) |
| `src/render3/` | `packages/compiler/render3/` | Ivy-specific codegen logic |
| `src/directive_matching.ts` | `packages/compiler/directivematching/` | Directive/component matching |
| `src/config.ts` | `packages/compiler/config/` | Configuration parsing |

## @angular/compiler-cli → Go packages

| TypeScript Path | Go Package | Purpose |
|---|---|---|
| `src/main.ts` | `packages/compiler-cli/cmd/ngc-go/` | CLI entry, tsconfig reading |
| `src/perform_compile.ts` | `packages/compiler-cli/compile/` | Main compilation loop |
| `src/ngtsc/` | `packages/compiler-cli/ngtsc/` | Ivy compiler logic (uses Node helper for types) |

## Node Helper

**Location**: `tools/ts-helper/` (npm package)

**Functions**:
- Extract component metadata (selector, template, templateUrl, inputs, outputs)
- Resolve templateUrl paths
- Provide TypeScript symbol info for type-checking
- Return JSON over stdout

**Called by**: Go code via `os/exec`

## Hybrid Approach

**Why?**
- TypeScript type information is complex; replicating tsc/tsserver in Go is substantial effort
- Template parsing logic is isolated and well-defined (easier to port)
- Node helper keeps tight integration with actual TypeScript compiler

**Go responsibilities**:
- Parse templates into AST
- Resolve template bindings (property, event, two-way)
- Generate factory functions (Ivy-compatible)
- Orchestrate compilation pipeline

**Node helper responsibilities**:
- Load TS files and extract `@Component` metadata
- Provide TypeScript symbol table
- Resolve module imports/dependencies
