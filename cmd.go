/*
TODO:
- dryrun
- result to stdout
- Cleanup variable names, find better ones
- Write some tests, make it testable
- Better duplicates check
- Sort array prior to output
- Create .gitignore file if not yet present
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	delimiterName            string = "goautogitignore"
	delimiterStartIdentifier string = "start"
	delimiterEndIdentifier   string = "end"
	comment                  string = "#"
	delimiterStart                  = "\n" + comment + " " + delimiterName + " " + delimiterStartIdentifier + "\n"
	delimiterEnd                    = comment + " " + delimiterName + " " + delimiterEndIdentifier + "\n"
)

var (
	flagSrcDir          *string = flag.String("dir", ".", "destination directory where .gitignore is located and where to traverse directory tree for go programs.")
	flagFindExecutables *bool   = flag.Bool("exec", false, "find all files with executable bit set")
	flagFindGoMain      *bool   = flag.Bool("gomain", true, "add executables, resulting from building go main packages")
)

var (
	srcdir      string
	executables []string
)

func insert(input string, addition string) (output string, err error) {
	if len(addition) == 0 {
		return input, nil
	}

	if !strings.HasSuffix(addition, "\n") {
		addition = addition + "\n"
	}
	addition = delimiterStart + addition + delimiterEnd
	if len(input) == 0 {
		return addition, nil
	}

	if strings.Contains(input, delimiterStart) {
		if strings.Count(input, delimiterStart) > 1 {
			return input, errors.New("multiple instances of start delimiter")
		}

		if strings.Contains(input, delimiterEnd) {
			if strings.Count(input, delimiterEnd) > 1 {
				return input, errors.New("multiple instances of closing delimiter")
			}
			if !strings.HasSuffix(input, "\n") {
				input = input + "\n"
			}

			startPos := strings.Index(input, delimiterStart)
			endPos := strings.Index(input, delimiterEnd) + len(delimiterEnd)

			output = input[:startPos] + addition + input[endPos:]

		} else {
			return input, errors.New("found no closing delimiter")
		}
	} else {
		if !strings.HasSuffix(input, "\n") {
			input = input + "\n"
		}
		output = input + addition
	}

	return output, nil
}

func main() {
	var err error

	log.SetFlags(0)
	flag.Parse()

	srcdir, err = filepath.Abs(filepath.Clean(*flagSrcDir))
	if err != nil {
		log.Fatalln(err)
	}

	fDstdir, err := os.Open(srcdir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalln(err)
		} else {
			log.Fatalln(err)
		}
	}
	defer fDstdir.Close()

	_, err = fDstdir.Readdir(1)
	if err != nil {
		log.Fatalln(srcdir, "is not a directory")
	}

	gitignore := srcdir + "/.gitignore"

	fGitignore, err := os.Open(gitignore)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalln(gitignore, "does not exists", err)
		} else {
			log.Fatalln(gitignore, "not readable", err)
		}
	}
	defer fGitignore.Close()

	gitignoreContentBytes, err := ioutil.ReadFile(gitignore)
	if err != nil {
		log.Fatalln(gitignore, "unable to read", err)
	}

	filepath.Walk(srcdir, walkTree)

	gitIgnoreExecutables, err := insert(string(gitignoreContentBytes), strings.Join(executables, "\n"))
	if err != nil {
		log.Fatalln("insert to gitignore failed:", err)
	}

	fmt.Println(gitIgnoreExecutables)
}

func walkTree(path string, info os.FileInfo, err error) error {
	// Skip .git directory tree, .gitignore and directories
	if strings.Contains(path, "/.git/") || strings.HasSuffix(path, ".gitignore") || info.IsDir() {
		return nil
	}

	var appendFile string

	// If -exec flag and file is executable
	if *flagFindExecutables && info.Mode()&0111 > 0 {
		exe, err := filepath.Rel(srcdir, path)
		if err != nil {
			fmt.Println("filepath.Rel", err)
			return nil
		}
		appendFile = exe
	}

	// If -gomain flag and file is go main
	if *flagFindGoMain && filepath.Ext(path) == ".go" {
		goContentBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return nil
		}

		if strings.Contains(string(goContentBytes), "package main\n") {
			fmt.Println("go main found:", path, filepath.Dir(path))
			dir := filepath.Dir(path)
			exec := dir[strings.LastIndex(dir, string(filepath.Separator))+1:]
			exe, err := filepath.Rel(srcdir, dir+string(filepath.Separator)+exec)
			if err != nil {
				fmt.Println("filepath.Rel", err)
				return nil
			}
			appendFile = exe
		}
	}

	if len(appendFile) > 0 {
		// Add file only once
		for _, exe := range executables {
			if exe == appendFile {
				return nil
			}
		}
		executables = append(executables, appendFile)
	}
	return nil
}