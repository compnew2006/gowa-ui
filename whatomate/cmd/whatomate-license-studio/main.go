package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/compnew2006/whatomate/internal/licenseissuer"
	"github.com/compnew2006/whatomate/internal/licensestudio"
)

func main() {
	defaultDataDir, err := licenseissuer.DefaultStudioDataDir()
	if err != nil {
		fatalf("resolve default data dir: %v", err)
	}

	fs := flag.NewFlagSet("whatomate-license-studio", flag.ExitOnError)
	addr := fs.String("addr", licenseissuer.DefaultStudioBindAddr, "Local bind address")
	dataDir := fs.String("data-dir", defaultDataDir, "Data directory for registry and key ring")
	noOpen := fs.Bool("no-open", false, "Do not open a browser automatically")
	_ = fs.Parse(os.Args[1:])

	logger := log.New(os.Stdout, "", log.LstdFlags)
	server, err := licensestudio.NewServer(licensestudio.Config{
		Addr:        *addr,
		DataDir:     *dataDir,
		OpenBrowser: !*noOpen,
		Logger:      logger,
	})
	if err != nil {
		fatalf("initialize license studio: %v", err)
	}

	if err := server.RunUntilSignal(); err != nil {
		fatalf("run license studio: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
