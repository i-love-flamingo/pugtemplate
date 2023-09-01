package pugjs

import (
	"sync"
)

type (
	// Startup controls the background processes needed on flamingo startup.
	// It is supposed to be used in a singleton as bound in module.go
	Startup struct {
		wg   sync.WaitGroup
		done bool
	}
)

// AddProcess adds and starts a background process concurrently
func (s *Startup) AddProcess(f func()) {
	s.wg.Add(1)
	go func() {
		f()
		s.wg.Done()
	}()
}

// Finish finishes up the whole Startup process. Must only be called after all AddProcess have been made
func (s *Startup) Finish() {
	var done = make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	go func() {
		<-done
		s.done = true
	}()
}

// IsFinished indicates if all startup processes have been finished
func (s *Startup) IsFinished() bool {
	return s.done
}
