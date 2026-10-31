package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"diskscope/internal/scan"
	"diskscope/internal/ui"
)

const version = "0.1.0"

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	var excludes stringList

	apparent := flag.Bool("apparent-size", false, "use apparent file size (st_size) instead of allocated blocks")
	oneFS := flag.Bool("one-file-system", true, "do not cross filesystem boundaries")
	topFiles := flag.Int("top-files", 500, "number of largest files to remember globally and per category")
	topPerDir := flag.Int("top-per-dir", 20, "number of small files to retain per directory")
	minSize := flag.String("min-file-size", "1MiB", "retain every file at least this large (suffixes: K, MiB, GB, ...)")
	workers := flag.Int("workers", runtime.NumCPU(), "scan worker goroutines")
	noColor := flag.Bool("no-color", false, "disable all color output")
	ascii := flag.Bool("ascii", false, "use ASCII glyphs instead of Unicode")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Var(&excludes, "exclude", "glob or absolute path to exclude (repeatable)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "diskscope %s - interactive disk usage explorer\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: diskscope [path] [flags]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("diskscope", version)
		return
	}
	if flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "diskscope: expected at most one path argument")
		flag.Usage()
		os.Exit(2)
	}

	root := ""
	if flag.NArg() == 1 {
		root = flag.Arg(0)
	} else if home, err := os.UserHomeDir(); err == nil {
		root = home
	} else {
		root = "."
	}

	minBytes, err := scan.ParseSize(*minSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "diskscope: %v\n", err)
		os.Exit(2)
	}
	if *topFiles < 1 || *topPerDir < 1 || *workers < 1 {
		fmt.Fprintln(os.Stderr, "diskscope: top-files, top-per-dir, and workers must be positive")
		os.Exit(2)
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "diskscope: %v\n", err)
		os.Exit(1)
	}
	if info, err := os.Stat(abs); err != nil {
		fmt.Fprintf(os.Stderr, "diskscope: %v\n", err)
		os.Exit(1)
	} else if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "diskscope: %s is not a directory\n", abs)
		os.Exit(1)
	}

	opts := scan.DefaultOptions(abs)
	opts.ApparentSize = *apparent
	opts.OneFileSystem = *oneFS
	opts.Excludes = excludes
	opts.TopFiles = *topFiles
	opts.TopPerDir = *topPerDir
	opts.MinFileSize = minBytes
	opts.Workers = *workers

	theme := ui.NewTheme(*noColor, *ascii)
	model := ui.NewModel(opts, theme)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "diskscope: %v\n", err)
		os.Exit(1)
	}
}
