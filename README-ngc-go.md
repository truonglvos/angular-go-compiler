# ngc-go — Angular Compiler in Go

A hybrid TypeScript/Go implementation of the Angular compiler for the latest Angular version (v18+).

## Architecture

- **Go packages** (`angular/packages/compiler/go/`, `angular/packages/compiler-cli/go/`): Template parsing, AST processing, code generation (Ivy factories).
- **Node helper** (`tools/ts-helper/`): TypeScript metadata extraction and symbol resolution via `tsserver`.
- **CLI** (`angular/packages/compiler-cli/go/cmd/`): Entry point `ngc-go` for compilation.

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

# Build Go CLI
cd ../../
go build -o bin/ngc-go ./angular/packages/compiler-cli/go/cmd
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
angular/
├── packages/
│   ├── compiler/                    # Original TypeScript @angular/compiler
│   │   ├── src/                     # TS source
│   │   └── go/                      # Go port
│   │       ├── mlparser/            # Template tokenization
│   │       ├── templateparser/      # Template AST & parser
│   │       ├── exprparser/          # Expression parser (bindings, events)
│   │       ├── output/              # Code generator
│   │       ├── render3/             # Ivy rendering logic
│   │       ├── directivematching/   # CSS selector matching for directives
│   │       └── config/              # Configuration loading
│   │
│   └── compiler-cli/                # Original TypeScript @angular/compiler-cli
│       ├── src/                     # TS source
│       └── go/                      # Go port
│           ├── cmd/                 # CLI entry (main.go)
│           ├── compile/             # Main compilation loop
│           └── ngtsc/               # Ivy-specific logic
│
tools/
└── ts-helper/                       # Node helper for metadata extraction
    ├── src/
    │   └── index.ts                 # Component metadata scanner
    ├── package.json
    └── tsconfig.json
```

## Development Roadmap

### Phase 1 (Current)
- Scaffold Go packages mirroring TS structure
- Implement Node helper for component metadata extraction
- Create basic CLI

### Phase 2
- Template parser (lexer + AST builder)
- Expression parser (property, event, two-way bindings)
- Template binding resolver

### Phase 3
- Type integration via Node helper
- Template type-checking (match bindings to component I/O)

### Phase 4
- Code generator (Ivy factory functions)
- AOT compilation pipeline

### Phase 5+
- Pure-Go type system (eliminate Node dependency)
- Watch mode, incremental compilation
- Compatibility with existing bundlers (webpack, esbuild)

## Hybrid Approach

This compiler uses a **hybrid model**:
- **Go**: Fast template parsing, AST transformation, code generation
- **Node**: Accurate TypeScript type information and metadata extraction

This allows:
1. Correctness: Leverage actual TypeScript compiler for type info
2. Performance: Native Go for CPU-intensive parsing/codegen
3. Compatibility: Output compatible with Angular's ecosystem

Over time, we may explore a pure-Go implementation by replicating TypeScript type system in Go.

## Testing

```bash
# Run Go tests (add later)
go test ./...

# Run Node helper tests (add later)
cd tools/ts-helper && npm test
```

## Contributing

See `docs/mapping-ts-to-go.md` for detailed mapping and design decisions.

