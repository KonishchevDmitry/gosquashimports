package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run(paths []string) error {
	for _, path := range paths {
		if err := processPath(path); err != nil {
			return err
		}
	}
	return nil
}

func processPath(path string) error {
	if info, err := os.Stat(path); err != nil {
		return err
	} else if !info.IsDir() {
		if filepath.Ext(path) != ".go" {
			return fmt.Errorf("%s: not a Go file", path)
		}
		return squashImports(path)
	}

	return filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := entry.Name()

		if entry.IsDir() && name == "vendor" {
			return filepath.SkipDir
		}

		if !entry.Type().IsRegular() || filepath.Ext(name) != ".go" {
			return nil
		}

		return squashImports(path)
	})
}

func squashImports(path string) error {
	fileSet := token.NewFileSet()

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	astFile, err := parser.ParseFile(fileSet, path, data, parser.ParseComments)
	if err != nil {
		return err
	} else if len(astFile.Imports) <= 1 {
		return nil
	}

	tokenFile := fileSet.File(astFile.Pos())
	importsStartLine := tokenFile.Line(astFile.Imports[0].Pos())
	importsEndLine := tokenFile.Line(astFile.Imports[len(astFile.Imports)-1].Pos())

	for _, comment := range astFile.Comments {
		startLine := tokenFile.Line(comment.Pos())
		endLine := tokenFile.Line(comment.End())

		// Don't try to optimize imports with comments - it's more sophisticated task
		if endLine == importsStartLine-1 ||
			startLine >= importsStartLine && startLine <= importsEndLine ||
			startLine == importsEndLine+1 {
			return nil
		}
	}

	var changed bool
	lines := strings.Split(string(data), "\n")

	for line := importsEndLine - 1; line > importsStartLine; line-- {
		if strings.TrimSpace(lines[line-1]) == "" {
			tokenFile.MergeLine(line)
			changed = true
		}
	}
	if !changed {
		return nil
	}

	return rewriteFile(fileSet, astFile, path)
}

func rewriteFile(fileSet *token.FileSet, astFile *ast.File, path string) (retErr error) {
	fmt.Printf("Rewriting %s...\n", path)

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	tempPath := path + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer func() {
		if err := tempFile.Close(); err != nil && retErr == nil {
			retErr = err
		}

		if tempPath != "" {
			if err := os.Remove(tempPath); err != nil && retErr == nil {
				retErr = err
			}
		}
	}()

	writer := bufio.NewWriter(tempFile)
	if err := format.Node(writer, fileSet, astFile); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	tempPath = ""

	return nil
}
