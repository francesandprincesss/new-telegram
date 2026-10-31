package readreceipt

type Service struct {
	repository Repository
	messages   MessageAccess
	events     Publisher
}

func NewService(repository Repository, messages MessageAccess, events Publisher) *Service {
	return &Service{repository: repository, messages: messages, events: events}
}

// TODO: Implement MarkRead. Reject attempts to mark the caller's own messages
// and publish an event only after persistence succeeds.
