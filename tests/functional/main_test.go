//go:build functional

package functional_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bruli-lab/myCalendar/tests/harness"
)

var ntfyURL string

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	cleanBinary, err := harness.BuildApp(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer cleanBinary()
	endpoint, terminate, err := harness.StartNtfy(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer terminate()
	ntfyURL = endpoint
	return m.Run()
}
