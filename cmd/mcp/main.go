package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	exitCodeOK = iota
	exitCodeErr
)

func init() {
	flag.Parse()
	flag.Usage = func() {
		fmt.Print(`usage: mc file ...`)
	}
}

func main() {
	os.Exit(run())
}

func printError(err error) {
	fmt.Fprintln(os.Stderr, err)
}

func run() int {
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return exitCodeErr
	}

	for _, arg := range args {
		if _, err := os.Stat(arg); err != nil {
			printError(err)
			return exitCodeErr
		}
	}

	if err := mcp(args); err != nil {
		printError(err)
		return exitCodeErr
	}

	return exitCodeOK
}

func mcp(args []string) error {
	existed := make(map[string]bool, len(args))
	for _, arg := range args {
		if existed[arg] {
			return fmt.Errorf("duplicat source %s", arg)
		}
		existed[arg] = true
	}

	f, err := ioutil.TempFile("", "mcp-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	for _, arg := range args {
		f.WriteString(arg + "\n")
	}
	f.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, f.Name())
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("abort copy: %s", err)
	}

	b, err := ioutil.ReadFile(f.Name())
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return fmt.Errorf("no destination files")
	}

	dests := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	destLen := len(dests)

	argsLen := len(args)
	var sources []string

	if argsLen > destLen {
		sources = args[:destLen]
	} else {
		sources = args
	}

	for i, s := range sources {
		d := dests[i]
		if d == "" || s == d {
			continue
		}

		info, err := os.Stat(s)
		if err != nil {
			return err
		}

		if err := copy(s, dests[i], info); err != nil {
			return err
		}
	}

	return nil
}

func copy(src, dest string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return link(src, dest)
	}
	if info.IsDir() {
		return dcopy(src, dest, info)
	}
	return fcopy(src, dest, info)
}

func dcopy(srcDir, destDir string, info os.FileInfo) error {
	if err := os.MkdirAll(destDir, 0775); err != nil {
		return err
	}
	defer os.Chmod(destDir, info.Mode())

	files, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		sd := filepath.Join(srcDir, f.Name())
		dd := filepath.Join(destDir, f.Name())

		if err := copy(sd, dd, f); err != nil {
			return err
		}
	}
	return nil
}

func fcopy(src, dest string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := os.Chmod(out.Name(), info.Mode()); err != nil {
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	fmt.Printf("copy %s to %s\n", src, dest)
	if _, err := io.Copy(out, s); err != nil {
		return err
	}
	return nil
}

func link(src, dest string) error {
	src, err := os.Readlink(src)
	if err != nil {
		return err
	}
	return os.Symlink(src, dest)
}
