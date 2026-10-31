package presence

type Service struct {
	repository Repository
	peers      PeerLookup
	events     Publisher
}

func NewService(repository Repository, peers PeerLookup, events Publisher) *Service {
	return &Service{repository: repository, peers: peers, events: events}
}

// TODO: Implement SetStatus. Clients may choose Online, Idle, or DND; Offline
// is derived exclusively from the absence of active event streams.
