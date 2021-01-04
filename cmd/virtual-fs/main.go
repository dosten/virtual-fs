package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
)

type Blob interface {
	Parent() Directory
	Name() string
}

type File interface {
	Blob
	Content() []byte
	Size() int
}

type file struct {
	parent  Directory
	name    string
	content []byte
	size    int
}

func NewFile(wd Directory, name string, content []byte) (File, error) {
	return &file{
		name:    name,
		content: content,
		parent:  wd,
	}, nil
}

func (f *file) Parent() Directory {
	return f.parent
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Content() []byte {
	return f.content
}

func (f *file) Size() int {
	return len(f.content)
}

type Directory interface {
	Blob
	Children() []Blob
	Add(child Blob)
}

type directory struct {
	mu       sync.Mutex
	parent   Directory
	name     string
	children []Blob
}

func NewDirectory(wd Directory, name string, children []Blob) (Directory, error) {
	return &directory{
		parent:   wd,
		name:     name,
		children: children,
	}, nil
}

func NewRoot() Directory {
	return &directory{name: ""}
}

func (d *directory) Parent() Directory {
	return d.parent
}

func (d *directory) Name() string {
	return d.name
}

func (d *directory) Children() []Blob {
	return d.children
}

func (d *directory) Add(child Blob) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.children = append(d.children, child)
}

type filesystem struct {
	mu   sync.Mutex
	tree Directory
	wd   Directory
}

func NewFilesystem() *filesystem {
	root := NewRoot()
	return &filesystem{
		tree: root,
		wd:   root,
	}
}

func (f *filesystem) CreateDirectory(name string, children []Blob) error {
	dir, err := NewDirectory(f.wd, name, children)
	if err != nil {
		return err
	}

	f.wd.Add(dir)

	return nil
}

func (f *filesystem) CurrentDirectory() {
	wd := f.wd

	var path string
	for {
		path = wd.Name() + "/" + path
		wd = wd.Parent()
		if wd == nil {
			break
		}
	}

	fmt.Println(path)
}

func (f *filesystem) List(recursive bool) {
	var printChilds func(w io.Writer, dir Directory, recursive bool, prefix string)

	printChilds = func(w io.Writer, dir Directory, recursive bool, prefix string) {
		for _, blob := range dir.Children() {
			name := blob.Name()
			if prefix != "" {
				name = prefix + "/" + name
			}

			switch b := blob.(type) {
			case File:
				fmt.Fprintf(w, "%s\t%s\t%s\n", "file", fmt.Sprintf("%dkb", b.Size()/1024), name)
				break
			case Directory:
				fmt.Fprintf(w, "%s\t%s\t%s\n", "dir", "-", name)
				if recursive {
					printChilds(w, b, recursive, name)
				}
				break
			}
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	fmt.Fprintln(w, "TYPE\tSIZE\tNAME")

	printChilds(w, f.wd, recursive, "")

	w.Flush()
}

func (f *filesystem) ChangeDirectory(path string) error {
	var wd Directory

	f.mu.Lock()
	defer f.mu.Unlock()

	// Set the root directory as the current working directory
	if path == "/" {
		f.wd = f.tree
		return nil
	}

	// If the path starts with a slash that means we have an absolute path
	// and we should start from the root dir instead of the working dir
	if strings.HasPrefix(path, "/") {
		wd = f.tree
	} else {
		wd = f.wd
	}

	// Let's chunk the path in order to find the new working directory including parent directories
	parts := strings.Split(strings.Trim(path, "/"), "/")

	for _, part := range parts {
	L:
		switch part {
		case ".":
			// A single dot means the current directory so let's ignore it
			continue
		case "..":
			// A double dot part means the parent directory so let's try to jump to it (if exists)
			if parent := wd.Parent(); parent != nil {
				wd = parent
			}
			break
		default:
			// Let's check for a children directory with a matching name and jump to it, otherwise throw and error
			for _, child := range wd.Children() {
				if v, isDir := child.(Directory); isDir && v.Name() == part {
					wd = v
					break L
				}
			}
			return fmt.Errorf("the directory '%s' does not exist", path)
		}
	}

	f.wd = wd

	return nil
}

func (f *filesystem) CreateFile(name string, content []byte) error {
	file, err := NewFile(f.wd, name, content)
	if err != nil {
		return err
	}

	f.wd.Add(file)

	return nil
}

var fs = NewFilesystem()

func ExecuteCommand(command string, args []string) error {
	var err error

	switch command {
	case "touch":
		name := args[0]
		err = fs.CreateFile(name, []byte{})
		break
	case "mkdir":
		name := args[0]
		err = fs.CreateDirectory(name, []Blob{})
		break
	case "ls":
		recursive := false
		for _, opt := range args {
			if opt == "-r" {
				recursive = true
				break
			}
		}
		fs.List(recursive)
		break
	case "pwd":
		fs.CurrentDirectory()
		break
	case "cd":
		path := args[0]
		err = fs.ChangeDirectory(path)
		break
	case "quit":
		os.Exit(0)
	default:
		err = fmt.Errorf("unknown command: %s", command)
		break
	}

	return err
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		scanner.Scan()
		text := scanner.Text()
		if len(text) != 0 {
			args := strings.Split(text, " ")
			err := ExecuteCommand(args[0], args[1:])
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}
