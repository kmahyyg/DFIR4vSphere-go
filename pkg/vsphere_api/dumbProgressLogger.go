package vsphere_api

import (
	"fmt"
	"github.com/vmware/govmomi/vim25/progress"
	"io"
	"os"
	"sync"
	"time"
)

type dumbProgressLogger struct {
	prefix string

	wg sync.WaitGroup

	sink chan chan progress.Report
	done chan struct{}
}

// Log outputs the specified string, prefixed with the current time.
// A newline is not automatically added. If the specified string
// starts with a '\r', the current line is cleared first.
func (p *dumbProgressLogger) Log(s string) (int, error) {
	if len(s) > 0 && s[0] == '\r' {
		p.Write([]byte{'\r', 033, '[', 'K'})
		s = s[1:]
	}

	return p.WriteString(time.Now().Format("[2006-01-02T15:04:05Z07:00] ") + s)
}

func (p *dumbProgressLogger) WriteString(s string) (int, error) {
	return p.Write([]byte(s))
}

func (p *dumbProgressLogger) Write(b []byte) (int, error) {
	w := os.Stdout
	if w == nil {
		w = os.Stdout
	}
	n, err := w.Write(b)
	if w == os.Stdout {
		os.Stdout.Sync()
	}
	return n, err
}

func newDumbProgressLogger(prefix string) *dumbProgressLogger {
	p := &dumbProgressLogger{
		prefix: prefix,

		sink: make(chan chan progress.Report),
		done: make(chan struct{}),
	}

	p.wg.Add(1)

	go p.loopA()

	return p
}

// loopA runs before Sink() has been called.
func (p *dumbProgressLogger) loopA() {
	var err error

	defer p.wg.Done()

	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	called := false

	for stop := false; !stop; {
		select {
		case ch := <-p.sink:
			err = p.loopB(tick, ch)
			stop = true
			called = true
		case <-p.done:
			stop = true
		case <-tick.C:
			line := fmt.Sprintf("\r%s", p.prefix)
			p.Log(line)
		}
	}

	if err != nil && err != io.EOF {
		p.Log(fmt.Sprintf("\r%sError: %s\n", p.prefix, err))
	} else if called {
		p.Log(fmt.Sprintf("\r%sOK\n", p.prefix))
	}
}

// loopA runs after Sink() has been called.
func (p *dumbProgressLogger) loopB(tick *time.Ticker, ch <-chan progress.Report) error {
	var r progress.Report
	var ok bool
	var err error

	for ok = true; ok; {
		select {
		case r, ok = <-ch:
			if !ok {
				break
			}
			err = r.Error()
		case <-tick.C:
			line := fmt.Sprintf("\r%s", p.prefix)
			if r != nil {
				line += fmt.Sprintf("(%.0f%%", r.Percentage())
				detail := r.Detail()
				if detail != "" {
					line += fmt.Sprintf(", %s", detail)
				}
				line += ")"
			}
			p.Log(line)
		}
	}

	return err
}

func (p *dumbProgressLogger) Sink() chan<- progress.Report {
	ch := make(chan progress.Report)
	p.sink <- ch
	return ch
}

func (p *dumbProgressLogger) Wait() {
	close(p.done)
	p.wg.Wait()
}
