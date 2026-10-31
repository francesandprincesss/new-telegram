package presence

// Manager will combine the persisted manual status with the number of active
// event streams for each user.
//
// TODO: Implement concurrency-safe 0→1 and 1→0 transitions. Multiple streams
// for the same user must not cause a premature Offline event.
type Manager struct{}
