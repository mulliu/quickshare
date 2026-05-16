package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/mulliu/quickshare/internal/fstore"
	"github.com/mulliu/quickshare/internal/netutil"
	"github.com/mulliu/quickshare/internal/qr"
	"github.com/mulliu/quickshare/internal/server"
)

var version = "0.1.0"

func main() {
	port := flag.Int("p", 0, "port to listen on")
	outputDir := flag.String("o", "./downloads", "output directory for received files")
	maxSize := flag.Int64("max-size", 4<<30, "max upload size in bytes (default: 4GB)")
	ttl := flag.Duration("ttl", 1*time.Hour, "file TTL before auto-cleanup (0 = no cleanup)")
	share := flag.String("s", "", "pre-share a file at startup")
	noBrowser := flag.Bool("n", false, "don't auto-open browser")
	flag.Parse()

	lanIP, err := netutil.FindLANIP()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var listener net.Listener
	preferred := []int{8080, 3000, 8000}
	if *port != 0 {
		preferred = []int{*port}
	}
	listener, *port, err = netutil.Listen(preferred...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	baseURL := fmt.Sprintf("http://%s:%d", lanIP, *port)
	localURL := fmt.Sprintf("http://127.0.0.1:%d", *port)

	qrPNG, err := qr.GeneratePNG(baseURL, 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	absDir, _ := filepath.Abs(*outputDir)
	store, err := fstore.New(absDir, *ttl, baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if *share != "" {
		f, err := os.Open(*share)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		store.Add(filepath.Base(*share), "", f)
		f.Close()
	}

	srv, err := server.New(store, lanIP, *port, *maxSize, qrPNG)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; srv.Shutdown(context.Background()) }()

	if !*noBrowser && runtime.GOOS == "windows" {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", localURL).Start()
	}

	srv.Serve(listener)
}
