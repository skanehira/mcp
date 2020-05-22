package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func compareDiff(t *testing.T, source, dest string) {
	t.Helper()

	_, err := os.Lstat(dest)
	if err != nil {
		t.Fatalf("invalid file %s: %s", dest, err)
	}

	s, err := ioutil.ReadFile(source)
	if err != nil {
		t.Fatalf("failed to read source file %s: %s", source, err)
	}

	d, err := ioutil.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read dest file %s: %s", dest, err)
	}

	if string(s) != string(d) {
		t.Fatalf("has diff\n  src:%s\n  dest:%s\n", string(s), string(d))
	}
}

func createFile(t *testing.T, name string) {
	t.Helper()

	dir := filepath.Dir(name)
	if _, err := os.Lstat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatalf("failed to create directory %s: %s", dir, err)
		}
	}

	f, err := os.Create(name)
	if err != nil {
		t.Fatalf("failed to create source file %s: %s", name, err)
	}
	f.WriteString("1234")
	f.Close()
}

func flatten(t *testing.T, f string) []string {
	t.Helper()

	info, err := os.Stat(f)
	if err != nil {
		t.Fatalf("failed to get info: %s", err)
	}
	if !info.IsDir() {
		return []string{f}
	}

	var files []string
	contents, err := ioutil.ReadDir(f)
	if err != nil {
		t.Fatalf("failed to get directory contents: %s", err)
	}
	for _, c := range contents {
		if c.IsDir() {
			files = append(files, flatten(t, filepath.Join(f, c.Name()))...)
		} else {
			files = append(files, filepath.Join(f, c.Name()))
		}
	}
	return files
}

func symlink(t *testing.T, src, dest string) {
	t.Helper()

	s, err := os.Readlink(src)
	if err != nil {
		t.Fatalf("failed get symblink: %s", err)
	}

	err = os.Symlink(s, dest)
	if err != nil {
		t.Fatalf("failed set symblink: %s", err)
	}
}

func TestMcpSuccess(t *testing.T) {
	stdout = ioutil.Discard

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(dir)

	for _, f := range []string{
		filepath.Join(dir, "file"),
		filepath.Join(dir, "sub", "file"),
	} {
		createFile(t, f)
	}

	err = os.Symlink(filepath.Join(dir, "file"), filepath.Join(dir, "symlink"))
	if err != nil {
		t.Fatalf("failed set symblink: %s", err)
	}

	tests := []struct {
		sources []string
		dests   []string
	}{
		{
			sources: []string{
				filepath.Join(dir, "file"),
				filepath.Join(dir, "sub"),
				filepath.Join(dir, "symlink"),
			},
			dests: []string{
				filepath.Join(dir, "dest", "file"),
				filepath.Join(dir, "dest", "sub"),
				filepath.Join(dir, "dest_symlink"),
			},
		},
	}

	for _, tt := range tests {
		if err := mcp(tt.sources, tt.dests); err != nil {
			t.Fatalf("failed to copy files: %s", err)
		}

		var sources, dests []string

		for _, s := range tt.sources {
			sources = append(sources, flatten(t, s)...)
		}

		for _, d := range tt.dests {
			dests = append(dests, flatten(t, d)...)
		}

		for i, s := range sources {
			compareDiff(t, s, dests[i])
		}
	}
}
