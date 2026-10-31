package reaction

type Service struct {
	repository Repository
	messages   MessageAccess
	events     Publisher
}

func NewService(repository Repository, messages MessageAccess, events Publisher) *Service {
	return &Service{repository: repository, messages: messages, events: events}
}

// TODO: Validate the reaction whitelist, verify dialog membership, persist the
// final state, and publish it to both participants after commit.
