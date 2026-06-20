package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"

	"veltryx/internal/logbuffer"
	"veltryx/internal/metrics"
	"veltryx/internal/monitoring"
	"veltryx/internal/tui"
)

var version = "dev"

func main() {
	serviceName := flag.String("service", "Veltryx WebSocket Ops", "service display name")
	useMock := flag.Bool("mock", true, "run with mock metrics feed")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logs := logbuffer.New(500)
	recorder := metrics.NewRecorder()
	monitor := monitoring.New(*serviceName, version, recorder, logs)

	if *useMock {
		mock := monitoring.NewMockFeed(recorder, logs)
		go mock.Run(ctx)
	}

	if !isTTY() {
		runPlain(ctx, monitor)
		return
	}

	if err := tui.Run(monitor); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd()))
}

func runPlain(ctx context.Context, monitor *monitoring.Monitor) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s := monitor.Snapshot()
			fmt.Printf(
				"[%s] service=%s status=%s uptime=%s ws_clients=%d ws_in=%.1f/s ws_out=%.1f/s x2=%s goroutines=%d\n",
				s.Service.Now.Format(time.RFC3339),
				s.Service.Name,
				s.Service.Status,
				s.Uptime().Truncate(time.Second),
				s.WS.ConnectedClients,
				s.WS.MessagesInPerSecond,
				s.WS.MessagesOutPerSecond,
				s.X2.Health,
				s.System.Goroutines,
			)
		}
	}
}
