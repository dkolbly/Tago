/*

 Tago "Emacs etags for Go"
 Author: Alex Combas
 Website: www.goplexian.com
 Email: alex.combas@gmail.com

 Version: 0.3
 © Alex Combas 2010
 © Manuel Odendahl 2012
 Initial release: January 03 2010

Added godoc1 compatibility, renamed annoying method names

 See README for usage, compiling, and other info.

*/

package main

import (
	"go/parser"
	"go/token"
	"go/ast"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"io"
)


func GetLine(r io.Reader, n int) (line []byte, err error) {
	var (
		newline byte = '\n'
		sought []byte
	)
	bufReader := bufio.NewReader(r)
	// iterate until reaching line #n
	for i := 1; i <= n; i++ {
		sought, err = bufReader.ReadBytes(newline)
		if err != nil {
			return
		}
	}
	line = sought[0:(len(sought) - 1)] //strip the newline
	return
}

// Returns the full line of source on which *ast.Ident.Name appears
func getFileLine(name string, n int) (line []byte, err error) {
	var file *os.File
	if file, err = os.OpenFile(name, os.O_RDONLY, 0666); err == nil {
		line, err = GetLine(file, n)
	}
	return
}

// Get working directory and set it for savePath flag default
func whereAmI() string {
	var r string = ""
	if dir, err := os.Getwd(); err != nil {
		fmt.Printf("Error getting working directory: %s\n", err)
	} else {
		r = dir + "/"
	}
	return r
}

// Setup flag variables
var saveDir = flag.String("d", whereAmI(), "Change save directory: -d=/path/to/my/tags/")
var tagsName = flag.String("n", "TAGS", "Change TAGS name: -n=MyTagsFile")
var appendMode = flag.Bool("a", false, "Append mode: -a")

// collect the tags for a file
type FileTags struct {
	bag bytes.Buffer
	files *token.FileSet
}

func (t *FileTags) String() string { return t.bag.String() }

func (t *FileTags) Write(p []byte) (n int, err error) {
	t.bag.Write(p)
	return len(p), nil
}

// Writes a TAGS line to a FileTags buffer
func (t *FileTags) tagIdent(leaf *ast.Ident) {
	pos := t.files.Position(leaf.Pos())
	if s, err := getFileLine(pos.Filename, pos.Line); err != nil {
		fmt.Println("Could not read line for ", leaf)
	} else {
		fmt.Fprintf(t, "%s%s%d,%d\n", s, leaf.Name, pos.Line, pos.Column)
	}
}

func (t *FileTags) parse(fileName string) (tags string, err error) {
	if ptree, perr := parser.ParseFile(t.files, fileName, nil, 0); perr != nil {
		return "", perr
	} else {
		// if there were no parsing errors then process normally
		for _, l := range ptree.Decls {
			switch leaf := l.(type) {
			case *ast.FuncDecl:
				t.tagIdent(leaf.Name)
			case *ast.GenDecl:
				for _, c := range leaf.Specs {
					switch cell := c.(type) {
					case *ast.TypeSpec:
						t.tagIdent(cell.Name)
					case *ast.ValueSpec:
						for _, atom := range cell.Names {
							t.tagIdent(atom)
						}
					}
				}
			}
		}
	}

	return t.String(), nil
}

type TagsFile FileTags

func (t *TagsFile) String() string { return (*FileTags)(t).String() }
func (t *TagsFile) Write(p []byte) (n int, err error) {
	return (*FileTags)(t).Write(p)
}

// TAGS file is either appended or created, not overwritten.
func (t *TagsFile) saveTags() {
	location := fmt.Sprintf("%s%s", *saveDir, *tagsName)
	if *appendMode {
		if file, err := os.OpenFile(location, os.O_APPEND|os.O_WRONLY, 0666); err != nil {
			fmt.Printf("Error appending file \"%s\": %s\n", location, err)
		} else {
			b := t.bag.Len()
			file.WriteAt(t.bag.Bytes(), int64(b))
			file.Close()
		}
	} else {
		if file, err := os.OpenFile(location, os.O_CREATE|os.O_WRONLY, 0666); err != nil {
			fmt.Printf("Error writing file \"%s\": %s\n", location, err)
		} else {
			file.WriteString(t.bag.String())
			file.Close()
		}
	}
}

// Parses the source files given on the commandline, returns a TAGS chunk for each file
func (tagFile *TagsFile) tagFiles(files []string) (err error) {
	tagFile.files = token.NewFileSet()

	for _, fileName := range(files) {
		t := new(FileTags)
		t.files = tagFile.files

		if tags, perr := t.parse(fileName); err != nil {
			return perr
		} else {
			totalBytes := len(tags)
			fmt.Fprintf(tagFile, "\f\n%s,%d\n%s", fileName, totalBytes, tags)
		}
	}

	return nil
}

func main() {
	flag.Parse()
	tea := new(TagsFile)
	tea.tagFiles(flag.Args())

	// if the string is empty there were parsing errors, abort
	if tea.String() == "" {
		fmt.Println("Parsing errors experienced, aborting...")
	} else {
		tea.saveTags()
	}
}
