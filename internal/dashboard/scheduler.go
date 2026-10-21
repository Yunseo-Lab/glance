package dashboard

import "time"

// StartScheduler keeps referenced sources warm without an open browser. It
// shares Read's persisted lease and global slots, rather than a second fetch path.
func (r *Runtime) StartScheduler() { r.startScheduler(time.Second) }
func (r *Runtime) startScheduler(interval time.Duration) {
	r.mu.Lock()
	if r.closed || r.schedulerStarted {
		r.mu.Unlock()
		return
	}
	r.schedulerStarted = true
	r.requireRegistered = true
	r.wg.Add(1)
	r.mu.Unlock()
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			sources, err := r.store.Sources()
			if err == nil {
				for _, source := range sources {
					select {
					case <-r.ctx.Done():
						return
					default:
					}
					_, _ = r.Read(source)
				}
			}
			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// CancelUnreferenced cancels HTTP work as soon as its final board reference is
// removed. The database generation/lease fence remains the authority on commit.
func (r *Runtime) CancelUnreferenced() {
	sources, err := r.store.Sources()
	if err != nil {
		return
	}
	active := map[string]bool{}
	for _, source := range sources {
		active[source.ID] = true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, task := range r.tasks {
		if task.running && !active[task.sourceID] && task.cancel != nil {
			task.cancel()
		}
	}
}

func (r *Runtime) CancelSource(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, task := range r.tasks {
		if task.running && task.sourceID == id && task.cancel != nil {
			task.cancel()
		}
	}
}
