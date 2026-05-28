package main

import (
	"flag"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type flags struct {
	listmodpack string
	filter      bool
	server      string
	client      string
	export      bool
}

// defines the global flags
func (f *flags) defineFlags() {
	flag.StringVar(&f.listmodpack, "listmodpack", "", "OPTION: <dir>\nLists mods in a modpack\n")
	flag.BoolVar(&f.filter, "f", false, "Enable filtering of results\n")
	flag.StringVar(&f.server, "server", "none",
		"OPTIONS: [required|optional|unsupported|unknown]\nfilter option for server side mods\n")
	flag.StringVar(&f.client, "client", "none",
		"OPTIONS: [required|optional|unsupported|unknown]\nfilter option for client side mods\n")
	flag.BoolVar(&f.export, "export", false, "Export list as json\n")
}

type Progress struct {
	mutex sync.RWMutex
	name  string
	done  chan struct{}

	now, max int
	ticker   *time.Ticker
}

func (p *Progress) Update(now int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.now = now
}

func (p *Progress) SetMax(max int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.max = max
}

func (p *Progress) Complete() {
	p.done <- struct{}{}
	close(p.done)
}

func (p *Progress) Start(t time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.ticker = time.NewTicker(t)
}

func newProgress(max int) *Progress {
	return &Progress{
		max:  max,
		done: make(chan struct{}),
	}
}

// Returns a string with the current progress over the max progress.
// eg. 6 / 77
func (p *Progress) ratio() string {
	return strconv.Itoa(p.now) + "/" + strconv.Itoa(p.max)
}

func progressRatio(cur, max int) string {
	return "(" + strconv.Itoa(cur) + "/" + strconv.Itoa(max) + ")"
}

func showCursor() {
	fmt.Print("\x1b[?25h")
}

func hideCursor() {
	fmt.Print("\x1b[?25l")
}

func cursorUp(n int) {
	fmt.Printf("\033[%dA", n)
}

func clearLine() {
	fmt.Print("\x1b[2k")
}

var success = greenIntenseColor
