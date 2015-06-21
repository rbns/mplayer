// Package mplayer provides facilities to run and control a mplayer instance.
// API is unstable for now.
package backend

import (
	"errors"
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Mplayer represents a mplayer process for playing one file.
// While mplayer supports playing more than one file from the commandline,
// this functionality is currently missing (but will maybe added when this
// library is overhauled).
type Mplayer struct {
	cmd    *exec.Cmd
	options []string
	file  string
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// Return a new Mplayer, ready to be started.
// options should be listed like on the commandline, eg.
// NewMplayer("foobar.mkv", []string{"-fs", "-alang", "hu,en"}...)
func NewMplayer(file string, options ...string) (*Mplayer, error) {
	_, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	m := new(Mplayer)
	m.file = file
	m.options = options
	
	return m, nil
}

// Prepare the exec.Cmd with all arguments
func (m *Mplayer)prepareCmd() error {
	if m.file == "" {
		return errors.New("no file to play")
	}
	args := make([]string, 2)
	
	// we always want to have these options set
	args[0] = "-quiet"
	args[1] = "-slave"
	args = append(args, m.options...)
	args = append(args, m.file)
	m.cmd = exec.Command("mplayer", args...)

	var err error
	m.stdin, err = m.cmd.StdinPipe()
	if err != nil {
		return err
	}

	m.stdout, err = m.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	m.stderr, err = m.cmd.StderrPipe()
	if err != nil {
		return err
	}

	return nil
}

// Actually start the mplayer instance.
func (m *Mplayer) start() error {
	var err error
	if m.running() {
		err = m.kill()
		if err != nil {
			return err
		}
	}
	
	if err := m.prepareCmd(); err != nil {
		return err
	}

	err = m.cmd.Start()
	if err != nil {
		return err
	}

	go func(c *exec.Cmd) {
		c.Wait()
		c.Process.Pid = 0
	}(m.cmd)

	return nil
}

// Check if an mplayer is running. This can't really check if the process
// is existing, because os.FindProcess is just a wrapper to create an
// os.Process with the given pid.. I'd be happy to know about better solutions.
func (m *Mplayer) running() bool {
	if m.cmd != nil {
		if m.cmd.Process != nil {
			if m.cmd.Process.Pid != 0 {
					return true
			}
		}
	}

	return false
}

// Kill a running mplayer process
func (m *Mplayer) kill() error {
	if m.running() {
		return m.cmd.Process.Kill()
	}
	return nil
}

// Get the value of an Mplayer-property.
// See http://www.mplayerhq.hu/DOCS/tech/slave.txt for a list of
// properties.
func (m *Mplayer) getProperty(name string) (string, error) {
	if !m.running() {
		return "", errors.New("mplayer isn't running")
	}

	err := m.sendCmd(fmt.Sprintf("pausing_keep get_property %v", name))
	if err != nil {
		return "", err
	}

	var ansName string
	var value string
	for {
		ansName, value, err = m.readAns()
		if err != nil {
			return "", err
		}

		if ansName == name {
			break
		}
	}

	return value, nil
}

// Read the answer for a getProperty request.
// This reads input lines, and returns the value of first line
// starting with "ANS_".
func (m *Mplayer) readAns() (string, string, error) {
	var line string
	var err error
	buf := bufio.NewReader(m.stdout)
	for {
		line, err = buf.ReadString('\n')
		if err != nil {
			return "", "", err
		}

		if strings.HasPrefix(line, "ANS_") {
			break
		}
	}

	splitted := strings.SplitN(line[4:len(line)-1], "=", 2)
	return splitted[0], splitted[1], nil
}

// Sends a command to stdin of mplayer.
func (m *Mplayer) sendCmd(name string, args ...string) error {
	if !m.running() {
		return errors.New("mplayer isn't running")
	}

	_, err := m.stdin.Write([]byte(fmt.Sprintf("%v %v\n", name, strings.Join(args, " "))))
	if err != nil {
		return err
	}
	return nil
}

// Sets a mplayer property.
// See http://www.mplayerhq.hu/DOCS/tech/slave.txt for a list of properties.
func (m *Mplayer) setProperty(name, value string) error {
	return m.sendCmd("pausing_keep set_property", name, value)
}

// Returns the path of the currently played file.
func (m *Mplayer) Path() (string, error) {
	return m.getProperty("path")

}

// Returns the length of the currently played file in seconds.
func (m *Mplayer) Length() (float64, error) {
	s, err := m.getProperty("length")
	if err != nil {
		return -1, err
	}

	return strconv.ParseFloat(s, 64)
}

// Returns the position in seconds in the currently played file.
func (m *Mplayer) Position() (float64, error) {
	s, err := m.getProperty("time_pos")
	if err != nil {
		return -1, err
	}

	return strconv.ParseFloat(s, 64)
}

// Is mplayer playing?
func (m *Mplayer) Playing() (bool) {
	return m.running()
}

// Start playback
func (m *Mplayer) Play() error {
	return m.start()
}

// Pause playback. Calling this again unpauses.
func (m *Mplayer) Pause() error {
	return m.sendCmd("pause")
}

// Stops playback
func (m *Mplayer) Stop() error {
	return m.kill()
}

// Seek a position in the file. Position in in seconds.
func (m *Mplayer) Seek(second int) error {
	return m.sendCmd("seek", fmt.Sprint(second), "2")
}
