# Portcullis — Project Steering

This is the engineering standard for this project. Every commit, every review, every decision
is measured against this. If something contradicts this doc, this doc wins.

---

## Technology Stack

- **Edge enforcement:** TypeScript (Lambda@Edge, V8 WASM evaluation)
- **Backend:** Go 1.26+ (poll, evaluate, admin Lambdas)
- **Policy:** Rego → WASM (OPA)
- **Infrastructure:** Terraform
- **Storage:** CloudFront KVS (verdicts), DynamoDB (policies, packages), S3 (WASM binaries)
- **Messaging:** SQS (re-evaluation fan-out), EventBridge (scheduling)
- **Testing (Go):** stdlib `testing` + testify for assertions, go-vcr for HTTP recording
- **Testing (TypeScript):** vitest, mocked AWS SDK
- **Linting (Go):** golangci-lint with C901 (complexity), `errcheck`, `govet`
- **Linting (TypeScript):** ESLint + strict tsconfig
- **Build (TypeScript):** esbuild (Lambda@Edge bundling)
- **CI:** GitHub Actions — lint, test, terraform validate, terraform plan on PR
- **Target:** AWS, self-hostable via `terraform apply`

---

## Coding Style

### Go

- `camelCase` for unexported, `PascalCase` for exported.
- Package names: short, lowercase, no underscores. No `util` packages.
- Errors are values. Check every `err` immediately after assignment. No discards without a comment.
- Return early on error. Happy path is unindented.
- Interfaces are small (1-3 methods). Accept interfaces, return structs.
- No `init()` functions. Explicit construction.
- Table-driven tests with descriptive subtest names.
- Context propagation: `ctx context.Context` as first parameter everywhere.
- Struct fields tagged for JSON where needed. No reflection-based magic.
- `internal/` for packages that aren't part of the public API.

### TypeScript

- Strict mode. No `any`. No `as` casts unless genuinely unavoidable (and commented why).
- `camelCase` for variables/functions, `PascalCase` for types/interfaces.
- Prefer `const` over `let`. No `var`.
- Pure functions where possible. Side effects at the boundary (handler entry point).
- Explicit return types on exported functions.
- Discriminated unions for result types (verdict, errors).
- No classes unless the AWS SDK forces it. Plain objects + functions.

### Terraform

- `snake_case` for everything.
- All variables must have a `description`.
- All outputs must have a `description`.
- Use `data` sources over hardcoded ARNs/IDs.
- No inline policies > 10 lines. Use `aws_iam_policy_document` data sources or `jsonencode` blocks.
- Modules have a README: what it creates, required inputs, assumptions.

---

## Functions

- If a function exceeds ~10 lines, it's doing more than one thing. Decompose.
- Exception: orchestration functions that sequence steps — but each step should be extracted.
- Pure functions for logic. Impure functions for I/O at the edges.
- One responsibility. If you can't name it without "and", split it.

---

## Error Handling

### Go

- Wrap errors with context: `fmt.Errorf("fetching vuln %s: %w", id, err)`.
- Custom error types for distinct failure modes the caller needs to handle differently.
- Never `panic` in library code. Only in main if setup is irrecoverable.
- Sentinel errors (`var ErrNotFound = errors.New(...)`) for expected conditions.
- `errors.Is` / `errors.As` for checking, not string matching.

### TypeScript

- Explicit result types over thrown exceptions for expected failures.
- Thrown exceptions for programmer errors (should never happen in production).
- Catch at the handler boundary, log, return appropriate HTTP/denial response.
- Never swallow errors silently.

---

## Testing

### Go

- Stdlib `testing` package. `testify` for assertions only (not suites).
- Table-driven tests with subtests (`t.Run`).
- `go-vcr` for recording and replaying HTTP interactions (OSV API, PyPI).
- Mocks via interfaces, not frameworks. Test doubles are explicit structs.
- Test error paths first. Happy path is obvious.
- `go test -race` in CI. Always.
- No test helpers that hide assertions. Each test reads top to bottom.

### TypeScript

- `vitest` with mocked AWS SDK and fetch.
- Test the handler with mocked KVS and enrichment services.
- Test WASM evaluation with real compiled policies.
- Test protocol adapters with URL-to-coordinate mappings.

### Both

- Isolated tests. No shared mutable state. No test ordering.
- Coverage proportional to blast radius. Policy evaluation and verdict logic need exhaustive tests. Terraform wiring does not.
- Test concurrency where relevant (Go: goroutine safety of shared state).

---

## Design Principles

- **Fail-closed.** When in doubt, deny. This is a security boundary.
- **Composition over inheritance.** Interfaces at boundaries, composed via constructors.
- **Protocol-agnostic core.** Everything below the adapter works on `PackageCoordinate`. The core doesn't know what PyPI is.
- **Make the state machine visible.** Package states (unknown → evaluating → allowed/denied) are explicit types, not strings.
- **Define once, derive outputs.** Rego policy compiles once to WASM. Evaluated at edge and in background. One source of truth.
- **No magic.** A dev should trace from CloudFront request to verdict without consulting framework docs.
- **Constrain toward correctness.** Strong types, exhaustive matching, explicit error handling.

---

## Documentation

- **Every exported function/type** gets a single-line comment explaining *why* it exists.
- **Comment *why*** when not obvious from the code. Never comment *what*.
- **ADRs** in `docs/decisions/`. Context, Decision, Consequences. One page max. Numbered chronologically.
- **README** shows the architecture, what's done, how to deploy. Landing page for a hiring manager.

---

## Git & Commits

- **Branch names:** `feature/<short-description>` or `fix/<short-description>`.
- **Commit messages:** Imperative mood, 50-char subject.
- **One logical change per commit.** Don't mix refactors with features.
- **No WIP commits on main.** Squash or rewrite before merging.
- **Tag releases** with semver, no v-prefix.

---

## Complexity Rules

- Function > 10 lines → decompose.
- File > 200 lines → question whether it's doing too many things.
- Cyclomatic complexity > 10 → golangci-lint flags it. Fix before committing.
- These are triggers to *think*, not hard limits.

---

## Self-Review Checklist

Before pushing:

1. **Go:** is every `err` checked immediately? No shadowing, no discards without comment.
2. **TypeScript:** strict mode passing? No `any` leaking in?
3. **Terraform:** every variable and output has a description?
4. Is every external call (OSV API, DynamoDB, KVS) wrapped in a timeout?
5. Are error cases tested, not just happy paths?
6. Does my addition meet this standard, even if surrounding code doesn't?
7. Would someone debug this at 2am without me?
8. Are linters passing clean? `golangci-lint run`, ESLint, `terraform validate`.
9. Can I explain every line if asked in an interview?

---

## How I Work On This

This project is for learning. AI does not write the code.

- **Explain** concepts, patterns, tradeoffs. Point me at docs.
- **Help** me debug when I'm stuck. Ask me what I've tried first.
- **Review** what I've written. Hold me to the standard in this doc.
- **Do not write code for me.** Not functions, not tests, not "here's a starting point."
- If I ask "how do I do X?" — explain the approach, show the API signature, link the docs. I type it.
- If I paste code and ask "is this right?" — review it, critique it, suggest improvements. Don't rewrite it.
- Scaffolding (build config, CI YAML, Terraform boilerplate) is an exception — that's plumbing, not learning.

The point is that I can explain every line because I wrote every line.

---

## What This Project Is Not

- Not a production service. Portfolio piece demonstrating distributed systems on AWS.
- Not a package management solution. Two policies, one protocol adapter, self-hostable.
- Not a UI project. Backend only.
- Not AI-generated slop. Every line understood, explainable, defensible.

---

## Quality Bar

> "Would this pass the bar for production at a company that does this for real?"

If the answer isn't yes, it's not done.
