//  Created on Sat Mar 30 2019
//
//  The MIT License (MIT)
//  Copyright (c) 2019 SummerPlay LLC
//
//  Permission is hereby granted, free of charge, to any person obtaining a copy of this software
//  and associated documentation files (the "Software"), to deal in the Software without restriction,
//  including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense,
//  and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so,
//  subject to the following conditions:
//
//  The above copyright notice and this permission notice shall be included in all copies or substantial
//  portions of the Software.
//
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED
//  TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
//  THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
//  TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package hatchery

import (
	"errors"
	"sync/atomic"
	"time"
)

var (
	// ErrAlreadyRunning is an error returned when a cron job is already running.
	ErrAlreadyRunning = errors.New("cron is already running")
)

// Executable is an executable process. Executables are executed in the background
// by CronJobs.
type Executable interface {
	// Execute start process exectuion. This is called in the background by a CronJob
	// on interval. The payload is passed to the executable's stdin. The output of the
	// executable is returned, along with any errors that occur during exectuion.
	Execute(payload []byte) ([]byte, error)
}

// CronJob executes an Executable in the background on interval until stoppped.
type CronJob struct {
	inverval    time.Duration
	executable  Executable
	runningFlag int32
	ticker      *time.Ticker
	errorCh     chan error
	outCh       chan []byte
}

// NewCronJob returns a new CronJob that will execute executable every interval.
// The provided payload is passed into the executable's stdin each time it is executed.
func NewCronJob(interval time.Duration, executable Executable) *CronJob {
	return &CronJob{
		inverval:   interval,
		executable: executable,
		errorCh:    make(chan error),
		outCh:      make(chan []byte),
	}
}

// Run begins the execution loop. The first execution will begin after the configured interval
// and repeat over and over every interval until Stop is called. ErrAlreadyRunning is returned
// if the CronJob is already running. This function is blocking, so it is usually called in a
// separate goroutine.
func (c *CronJob) Run() error {
	if !atomic.CompareAndSwapInt32(&c.runningFlag, 0, 1) {
		return ErrAlreadyRunning
	}
	c.ticker = time.NewTicker(c.inverval)
	for range c.ticker.C {
		go func() {
			b, err := c.executable.Execute(nil)
			if err != nil {
				c.errorCh <- err
				return
			}
			if b != nil {
				c.outCh <- b
			}
		}()
	}
	return nil
}

// Stop stops the cron loop. If an execution is already underway, it will still finish in the background,
// but no further exectuions will occur.
func (c *CronJob) Stop() {
	if atomic.CompareAndSwapInt32(&c.runningFlag, 1, 0) {
		c.ticker.Stop()
	}
}

// Errors returns a channel for listening for errors returned by the executable on execution.
// This channel is unbuffered, so it should be aggressively consumed.
func (c *CronJob) Errors() <-chan error {
	return c.errorCh
}

// Output returns a channel for listening for output from the executable on execution.
// This cahnnel is unbuffered, so it should be aggressively consumed.
func (c *CronJob) Output() <-chan []byte {
	return c.outCh
}
