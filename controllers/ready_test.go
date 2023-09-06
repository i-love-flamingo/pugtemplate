package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flamingo.me/pugtemplate/controllers"
	"flamingo.me/pugtemplate/pugjs"
	"github.com/stretchr/testify/assert"
)

func TestReady_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		process        func(testing.TB, *pugjs.Startup)
		wantStatusCode int
	}{
		{
			name:           "finished startup returns 200",
			process:        waitForFinish,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "unfinished startup returns 425",
			process:        func(testing.TB, *pugjs.Startup) {},
			wantStatusCode: http.StatusTooEarly,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := new(pugjs.Startup).Inject()

			tt.process(t, s)

			rec := httptest.NewRecorder()
			r := new(controllers.Ready).Inject(s)
			r.ServeHTTP(rec, &http.Request{})

			assert.Equal(t, tt.wantStatusCode, rec.Result().StatusCode)
		})
	}
}

func waitForFinish(t testing.TB, startup *pugjs.Startup) {
	t.Helper()

	startup.Finish()

	timeout := time.NewTimer(time.Second)
	t.Cleanup(func() { timeout.Stop() })

	ticker := time.NewTicker(10 * time.Millisecond)
	t.Cleanup(ticker.Stop)

	for {
		if startup.IsFinished() {
			return
		}

		select {
		case <-timeout.C:
			t.Fatal("timeout on waiting for Finish reached")
		case <-ticker.C:
			continue
		}
	}
}
