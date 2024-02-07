package gryph_test

import (
	"testing"

	"github.com/superloach/gryph"
)

func TestHelloWorld(t *testing.T) {
	py, err := gryph.NewInterp()
	if err != nil {
		t.Error(err)
	}

	if err := py.Start(); err != nil {
		t.Error(err)
	}

	out, err := py.Run("print('hello, world!')")
	if err != nil {
		t.Error(err)
	}

	if out != "hello, world!\n" {
		t.Errorf("unexpected output: %q", out)
	}

	if err := py.Close(); err != nil {
		t.Error(err)
	}
}
