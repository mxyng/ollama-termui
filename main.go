package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	bbt "github.com/charmbracelet/bubbletea/v2"

	"github.com/mxyng/ollama-termui/chat"
)

func main() {
	var baseUrl string
	flag.StringVar(&baseUrl, "base-url", "http://127.0.0.1:11434", "ollama host address")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage:", filepath.Base(os.Args[0]), "MODEL")
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if err := os.Truncate("debug.log", 0); err != nil {
		panic(err)
	}

	f, err := bbt.LogToFile("debug.log", "debug")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if _, err := bbt.NewProgram(chat.New(baseUrl, flag.Arg(0))).Run(); err != nil {
		panic(err)
	}
}
