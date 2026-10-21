package dashboard

import (
	"errors"
	"log"
)

var ErrNotFound = errors.New("대상을 찾을 수 없습니다")

// DashboardService owns application operations. HTTP is an adapter, not the
// owner of persistence, validation or source-refresh policy.
type DashboardService struct {
	providers *Providers
	store     *Store
	runtime   *Runtime
}

func (s *DashboardService) Board(id string) (Board, error) {
	return s.store.GetBoard(id)
}
func (s *DashboardService) Commit(board Board) (Board, error) {
	saved, err := s.store.Save(board)
	if err != nil {
		return Board{}, err
	}
	if s.runtime != nil {
		s.runtime.CancelUnreferenced()
	}
	if err = s.store.pruneSnapshots(saved.Sources); err != nil {
		log.Print("dashboard cache prune failed")
	}
	return saved, nil
}
func (s *DashboardService) Snapshots(id string) (map[string]Snapshot, error) {
	board, err := s.Board(id)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Snapshot, len(board.Sources))
	for _, source := range board.Sources {
		snapshot, err := s.runtime.Read(source)
		if err != nil {
			return nil, err
		}
		out[source.ID] = snapshot
	}
	return out, nil
}
