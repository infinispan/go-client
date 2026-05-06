# Pull Request Reviewing Instructions

GitHub pull requests should use the .github/pull_request_template.md file which contains user instructions for
submitting well-formed pull requests. Ensure that the pull request follows the rules, when they are applicable.

## Verify the author checklist

The PR template includes an author checklist. Verify that checklist items are actually satisfied, not just checked off.
Pay particular attention to:
- Test coverage: if tests are missing, is the justification convincing?
- Documentation updates: if missing, is the justification convincing?
- Commit messages reference a GitHub issue in the `[#00000] Summary` format.

## What Important means here

Reserve Important for findings that would break behavior, leak data, or block a rollback: incorrect logic,
PII in logs or error messages, and protocol incompatibilities. Style, naming, and refactoring suggestions are Nit at most.

## Infinispan Go client-specific concerns

Watch for these domain-specific issues that general review might miss:
- **Wire protocol compatibility:** changes to operation encoding/decoding must remain compatible with the Hot Rod 4.1 protocol specification. Verify opcodes, field ordering, and LP-bytes/VInt encoding.
- **ProtoStream compatibility:** WrappedMessage encoding must match the Java ProtoStream format. Changes to field numbers or wrapping logic can silently break interop with the server.
- **Thread safety:** shared state (connection pool, LRU cache, bloom filter counters) is accessed from multiple goroutines — verify that new mutable shared state is properly guarded with `sync.Mutex` or `sync/atomic`.
- **Goroutine lifecycle:** goroutines started by the library must be tied to a `done` channel or context for clean shutdown. Leaking goroutines is a blocking issue.
- **Public API surface:** the `hotrod/` package is the public API. New exported types, functions, or method signatures are hard to change later — verify they follow existing conventions (functional options, `[]byte` keys/values, context-first parameters).
- **`internal/` boundary:** nothing outside the module should import `internal/` packages except `test/` for integration tests.

## Cap the nits

Report at most five Nits per review. If you found more, say "plus N similar items" in the summary instead of posting
them inline. If everything you found is a Nit, lead the summary with "No blocking issues."

## Do not report

- Anything CI already enforces: `go vet`, formatting, type errors
- Test-only code that intentionally violates production rules

## CI Failures

- If the CI checks have been executed, look at the results and try to determine if any failures are related to the changes.
