package rootdb

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Adjust working dir to match what our services expect.
	os.Chdir("../..")
	os.Exit(m.Run())
}
