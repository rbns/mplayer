package backend

import (
	"testing"
	"time"
)

func TestAll(t *testing.T) {
	m, err := NewMplayer("test.webm")
	if err != nil {
		t.Error(err)
	}
	
	err = m.Play()
	if err != nil {
		t.Error("play", err)
	}

	// i know, time shouldn't be used in tests. i'll look for a better way
	// to assure that the mplayer process has started.
	time.Sleep(100 * time.Millisecond)

	err = m.Pause()
	if err != nil {
		t.Error("pause", err)
	}

	time.Sleep(5000 * time.Millisecond)

	err = m.Pause()
	if err != nil {
		t.Error("pause2", err)
	}

	err = m.Stop()
	if err != nil {
		t.Error("stop", err)
	}

}
