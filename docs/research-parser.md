# Research: TypeScript / Angular parsing options for ngc-go

Goal: choose an approach to obtain AST + type information from TypeScript sources and Angular component templates, from a Go-based compiler.

Options and tradeoffs

- Node TypeScript (tsc / tsserver)
  - Pros: Official TypeScript compiler + language service, full type information and tooling, supports project references, emits JSON/AST via APIs.
  - Cons: Requires Node.js runtime; integration from Go needs IPC (spawn child process) or an RPC layer.

- Tree-sitter (TypeScript grammar)
  - Pros: Fast incremental parsing, can be used from Go via `go-tree-sitter` bindings; good for syntax extraction without Node dependency.
  - Cons: No type information; grammar may lag TS features; requires C bindings and build steps.

- SWC / other parsers (Rust-based)
  - Pros: Fast, supports modern JS/TS; can be driven via CLI or embedding a small service.
  - Cons: Also external dependency; type information limited compared to TypeScript language service.

- Hybrid approach (recommended for V1)
  - Use the TypeScript language service (`tsserver`) for accurate type info and diagnostics.
  - Use pure-Go parsing (Tree-sitter or simple regex/AST) for template extraction when feasible to keep Go code native.
  - Start with a Node-based helper (small npm package) that exposes necessary information (component metadata, template text, type symbols) over JSON-RPC or stdout JSON. Replace later with a pure-Go parser if desired.

Recommendation

For correctness and faster progress on V1, use a hybrid approach: call `tsserver`/a small Node helper to get reliable AST and type information, while keeping template parsing and codegen implemented in Go. This gives accurate type-checking and faster feature parity with Angular's expectations.

Next actions
- Prototype a Node helper that, given a project path, returns a list of components with `selector`, `template` or `templateUrl`, and TypeScript symbol info.
- Add a Go wrapper that spawns the helper and consumes JSON output.
- Optionally evaluate `go-tree-sitter` for local parsing of templates only.
