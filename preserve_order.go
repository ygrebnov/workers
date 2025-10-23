package workers

// Reorderer (preserve-order coordinator)
//
// Responsibility:
// - Consume completion events from workers and emit results strictly in the original
//   input order, regardless of completion order.
// - Advance the output cursor when a task has no result to emit (SendResult == false)
//   or when a task errored (present == false), so later tasks can flow.
//
// Inputs:
// - events <-chan completionEvent[R]: stream of task completion notifications produced by workers.
//   Each event carries:
//     - idx: input index assigned at AddTask time,
//     - id: optional task identifier (for observability only),
//     - val: the result value (when present == true),
//     - present: whether this completion has an output value to emit.
// - results chan<- R: outward results channel owned by Workers and written by the reorderer.
//
// Dependencies:
// - Internal state only: an integer cursor `next` that tracks the next expected index,
//   and two in-memory structures to buffer out-of-order completions:
//     - buf: map[int]R to store results received ahead of the cursor,
//     - seenNoRes: set[int] to remember indices that completed without a result.
// - No external services. The coordinator uses simple bookkeeping and channel operations.
//
// Semantics:
// - For each incoming completion event:
//     - If present == true, store val at buf[idx].
//     - If present == false, remember idx in seenNoRes.
//     - Then repeatedly flush from the cursor `next` while either:
//         a) a buffered result exists at buf[next] (emit results[next], delete, next++), or
//         b) an entry exists in seenNoRes for `next` (delete, next++),
//       stopping when neither condition holds.
// - On events channel close, perform a best-effort final flush of any contiguous tail
//   using the same flushing rules, then return. The reorderer never closes results itself.
//
// Edge cases:
// - Missing results: when tasks legitimately opt-out of SendResult, an event with present == false
//   must still advance the cursor so later results are not blocked.
// - Out-of-order completions: results are buffered until all prior indices have either emitted
//   or confirmed no-result; only then they are forwarded.
// - StopOnError + preserve-order: errored tasks are represented as present == false events
//   (no output), which correctly advance the cursor without emitting a value.
// - Shutdown behavior: if events close while there are gaps (missing earlier completions),
//   only the contiguous prefix from the current cursor can be flushed; remaining holes are not emitted.
//
// Concurrency contracts:
// - Single goroutine: the reorderer runs as one dedicated goroutine reading `events` and writing
//   to `results`. It does not require external synchronization.
// - Channel ownership: the reorderer only reads from `events` and writes to `results`; it never
//   closes either channel. Channel closure is orchestrated by Workers lifecycle.
// - Backpressure: writes to `results` are synchronous and respect the caller-provided buffer size.
//   The coordinator does not spawn extra goroutines.

// completionEvent represents a task completion used by the preserve-order coordinator.
// present == true means val contains a result to emit;
// false means no result (either SendResult==false or task errored).
// idx is the input index assigned at AddTask; id is carried for observability.
type completionEvent[R any] struct {
	idx     int
	id      any
	val     R
	present bool
}
