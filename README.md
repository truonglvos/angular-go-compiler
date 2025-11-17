# ngc-go — Angular Compiler in Go

A hybrid TypeScript/Go implementation of the Angular compiler for the latest Angular version (v18+).

## Architecture

- **Go packages** (`compiler/`, `compiler-cli/`): Template parsing, AST processing, code generation (Ivy factories).
- **Node helper** (`tools/ts-helper/`): TypeScript metadata extraction and symbol resolution via `tsserver`.
- **CLI** (`compiler-cli/cmd/ngc-go/main.go`): Entry point for compilation.

## Quick Start

### Prerequisites

- Go 1.20+
- Node.js 18+
- TypeScript 5.0+

### Build

```bash
# Set up Node helper
cd tools/ts-helper
npm install
npm run build
cd ../../

# Build Go CLI
go build -o bin/ngc-go ./packages/compiler-cli/cmd/ngc-go
```

### Run

```bash
# Show help
./bin/ngc-go help

# Compile a project
./bin/ngc-go compile /path/to/angular/project
```

## Directory Structure

```
.
├── packages/                        # Go ports of Angular packages
│   ├── compiler/                    # Port of @angular/compiler
│   │   ├── mlparser/                # Template tokenization
│   │   ├── templateparser/          # Template AST & parser
│   │   ├── exprparser/              # Expression parser (bindings, events)
│   │   ├── output/                  # Code generator
│   │   ├── render3/                 # Ivy rendering logic
│   │   ├── directivematching/       # CSS selector matching
│   │   └── config/                  # Configuration loading
│   │
│   └── compiler-cli/                # Port of @angular/compiler-cli
│       ├── cmd/ngc-go/              # CLI entry
│       ├── compile/                 # Main compilation loop
│       └── ngtsc/                   # Ivy-specific logic
│
├── angular/                         # Original Angular repo (reference)
│   └── packages/
│
├── tools/ts-helper/                 # Node helper for metadata extraction
│
├── docs/
│   └── mapping-ts-to-go.md
│
└── README.md
```

## Development Roadmap

### Phase 1 (Current)
- Scaffold Go packages
- Node helper for component metadata
- Basic CLI

### Phase 2
- Template parser (lexer + AST)
- Expression parser (bindings)

### Phase 3
- Type integration via Node helper
- Template type-checking

### Phase 4
- Code generator (Ivy factories)
- AOT pipeline

### Phase 5+
- Pure-Go type system
- Watch mode, incremental compilation

## Hybrid Approach

**Go**: Template parsing, AST transformation, code generation
**Node**: TypeScript type information and metadata extraction

This enables correctness, performance, and compatibility with Angular's ecosystem.

