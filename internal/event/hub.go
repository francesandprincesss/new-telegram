package event

// Hub will own in-memory subscriptions grouped by user and connection ID.
//
// TODO: Implement bounded per-connection queues, snapshot publication outside
// locks, idempotent unsubscribe, and slow-consumer disconnection. Never call a
// gRPC stream Send method while holding the Hub mutex.
type Hub struct{}
