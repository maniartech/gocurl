# AGENTS.md


## Core Principles

### Best-in-Industry Engineering Standard
- Build as if this code will become the team reference implementation.
- Prefer correctness, clarity, maintainability, operability, and safety over novelty.
- New code must be production-ready on first merge: documented, tested, lint-clean, and observable.
- APIs, package boundaries, and module contracts must feel deliberate and stable.
- Leave every touched file better than you found it.

### KISS — Keep It Simple, Stupid
- Write the simplest code that correctly solves the current problem.
- Prefer explicit data flow over clever abstractions.
- Flatten nesting where practical.
- If a comment is needed to explain *what* code does, rewrite the code.
- Do not introduce a new abstraction without a concrete, current need.

### DRY — Don't Repeat Yourself
- Every rule, error message, config key, and business invariant should have one canonical definition.
- Extract shared helpers only when duplication is stable and proven.
- Reuse existing modules before creating new ones.
- Do not copy logic between `pkg/`, `services/`, and `utils/`.

### SRP — Single Responsibility Principle
- Each package, file, type, and function should have one reason to change.
- Split long or mixed-responsibility functions early.
- Keep module public APIs minimal and cohesive.
- Avoid packages that mix transport, business logic, storage, and validation concerns.

### YAGNI — You Aren't Gonna Need It
- Do not add extension points, feature flags, or optional layers without an active consumer.
- Delete dead code instead of preserving it “just in case”.
- Avoid speculative interfaces and premature generalization.

### Military-Grade Robustness
- Handle every error.
- Validate all external input: config, env vars, HTTP payloads, file input, query input, and provider responses.
- Release resources deterministically with `defer` where appropriate.
- Concurrency must be intentional, bounded, and documented.
- Recover from panics only at process boundaries; log enough context to debug the failure.
- All code should be secure, thread-safe where required, and resilient under load.

### Test-Driven Development

- Write tests before writing the implementation.
- Tests must be deterministic, isolated, and fast.
- Cover all edge cases and error paths, not just the happy path. Ensure near 100% code coverage.
- Use table-driven tests for functions with multiple input cases.
- Use mocks and fakes to isolate units under test, but prefer real dependencies for integration tests.
- Test code should be as clean and maintainable as production code. Refactor test code to remove duplication and improve readability.
- Use descriptive test names that clearly indicate the scenario being tested and the expected outcome.
- Run tests frequently during development to catch regressions early. All tests must pass before merging code.
- Ensure -race is used for tests involving concurrency to catch data races.
- Use code coverage tools to identify untested code paths and add tests as needed to achieve comprehensive coverage.
- Continuously review and update tests as the codebase evolves to ensure they remain relevant and effective at catching bugs.

---

## Copilot Todos

Ensure that after finishing every todo item, the resulting code adheres to the above principles. If you find yourself writing code that violates any of these rules, stop and refactor until it complies before moving on to the next item.

Ensure that all code you write is idiomatic Go, following the best practices outlined in the next section. If you are unsure about how to implement something in Go, consult the official documentation or ask for clarification.

Ensure that after finishing every todo item, the code is fully tested with unit tests that cover all edge cases and error paths. If you find yourself writing code that is difficult to test, refactor it until it is easily testable.

Ensure that after finishing every todo item, the code is well-documented with clear comments explaining the purpose of each function, type, and package. If you find yourself writing code that is not self-explanatory, add comments or refactor it until it is clear.
