package pugjs_test

import (
	"testing"
	"time"

	"flamingo.me/pugtemplate/pugjs"
	"github.com/stretchr/testify/assert"
)

func TestStartup_Finish(t *testing.T) {
	t.Parallel()
	s := new(pugjs.Startup).Inject()

	wait := make(chan struct{})

	// add 2 processes
	for i := 0; i < 2; i++ {
		s.AddProcess(func() {
			<-wait
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
	ticker := time.NewTicker(10 * time.Millisecond)
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

}
