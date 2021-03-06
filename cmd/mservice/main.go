package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

// grep 'timer process' log.log | cut -d ' ' -f 5 | cut -d . -f 1 | grep -E '^[0-9]+$' > mics.csv

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
}

func main() {

	var (
		httpPort = flag.String("h", ":8080", "Control plane (HTTP) service port")
		tcpPort  = flag.String("t", ":8081", "Data plane (TCP) service port")
		repl     = flag.Bool("r", false, "REPL")
		specDir  = flag.String("s", DefaultSpecDir, "specs directory")
		libDir   = flag.String("i", "../..", "directory containing 'interpreters'")
		ttl      = flag.Duration("e", 60*time.Second, "crew cache TTL (0 to disable)")
	)

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// For fun, we'll watch all routed message here.
	routed := make(chan interface{}, 8)
	go func() {
	LOOP:
		for {
			select {
			case <-ctx.Done():
				break LOOP

			case message := <-routed:
				fmt.Printf("%s\n", JS(message))
			}
		}
	}()

	s, err := makeDemoService(ctx, routed, *specDir, *libDir)
	if err != nil {
		panic(err)
	}

	if 0 < *ttl {
		s.crewCache = NewCrewCache(*ttl, 1024)
	}

	if *httpPort != "" {
		go func() {
			if err = s.HTTPServer(ctx, *httpPort); err != nil {
				panic(err)
			}
		}()
	}

	if *repl {
		go func() {
			in := bufio.NewReader(os.Stdin)
			if err = s.Listener(ctx, in, os.Stdout, nil); err != nil {
				log.Printf("REPL: %s", err)
			}
			os.Exit(0)
		}()
	}

	if err = s.TCPListener(ctx, *tcpPort); err != nil {
		panic(err)
	}

	log.Printf("main terminating")
}
