package workers

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
