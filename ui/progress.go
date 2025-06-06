package ui

import (
	"fmt"
	"time"
)

type ProgressBar struct {
	ticker *time.Ticker
	done   chan bool
}

func NewProgressBar() *ProgressBar {
	return &ProgressBar{ticker: time.NewTicker(1 * time.Second), done: make(chan bool)}
}

func (p *ProgressBar) Start() {
	go func() {
		for {
			select {
			case <-p.ticker.C:
				fmt.Print(" 🟩")
			case <-p.done:
				return
			}
		}
	}()
}

func (p *ProgressBar) Increment() {
	fmt.Print(" ░")
}

func (p *ProgressBar) Stop() {
	p.ticker.Stop()
	p.done <- true
}
