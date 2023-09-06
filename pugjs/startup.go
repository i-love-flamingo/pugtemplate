package pugjs

import (
	"golang.org/x/sync/errgroup"
)

type (
	// Startup controls the background processes needed on flamingo startup.
	// It is supposed to be used in a singleton as bound in module.go
	Startup struct {
		eg   *errgroup.Group
		done chan struct{}
	}
)

// Inject dependencies
func (s *Startup) Inject() *Startup {
	s.eg = new(errgroup.Group)
	s.done = make(chan struct{})

	return s
}

// AddProcess adds and starts a background process concurrently
func (s *Startup) AddProcess(f func() error) {
	s.eg.Go(f)
}

// Finish finishes up the whole Startup process. Must only be called after all AddProcess have been made
// Finish does not block. Use IsFinished to check if the process is finished or the returned error channel to check if
// at least one of the processes returned an error
func (s *Startup) Finish() <-chan error {
	errChan := make(chan error)
	go func() {
		err := s.eg.Wait()
		if err != nil {
			errChan <- err
		}

		close(s.done)
		close(errChan)
	}()

	return errChan
}

// IsFinished indicates if all startup processes have been finished
func (s *Startup) IsFinished() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}
