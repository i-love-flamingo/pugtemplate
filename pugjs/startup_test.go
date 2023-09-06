package pugjs_test

import (
	"errors"
	"testing"
	"time"

	"flamingo.me/pugtemplate/pugjs"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestStartup_Finish(t *testing.T) {
	t.Parallel()

	t.Run("check if process finishes", func(t *testing.T) {
		t.Parallel()

		goleak.VerifyNone(t, goleak.IgnoreCurrent())

		s := new(pugjs.Startup).Inject()

		wait := make(chan struct{})

		// add 2 processes
		for i := 0; i < 2; i++ {
			s.AddProcess(func() error {
				<-wait

				return nil
			})
		}

		s.Finish()
		assert.False(t, s.IsFinished(), "already finished while processes still running")

		// finish 1 process should not finish the startup
		wait <- struct{}{}
		assert.False(t, s.IsFinished(), "already finished while processes still running")

		// finish 2nd process
		wait <- struct{}{}

		// finish state is updated async, so we retry a bit with timeout
		timeout := time.NewTimer(5 * time.Second)
		t.Cleanup(func() { timeout.Stop() })

		ticker := time.NewTicker(10 * time.Millisecond)
		t.Cleanup(ticker.Stop)

		for {
			if s.IsFinished() {
				break
			}

			select {
			case <-timeout.C:
				t.Fatalf("timeout on wating for finish")
			case <-ticker.C:
				continue
			}
		}
	})

	t.Run("error in process is passed via channel", func(t *testing.T) {
		t.Parallel()

		goleak.VerifyNone(t, goleak.IgnoreCurrent())

		s := new(pugjs.Startup).Inject()

		s.AddProcess(func() error {
			return errors.New("some error")
		})

		errs := s.Finish()
		select {
		case err := <-errs:
			assert.Error(t, err)
		case <-time.After(time.Second):
			t.Errorf("timeout on wating for finish")
		}
	})

}
