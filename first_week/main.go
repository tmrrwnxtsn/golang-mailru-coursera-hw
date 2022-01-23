package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := "."
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	files, err := file.Readdir(-1)
	if err != nil {
		return err
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	if printFiles {
		for idx, fileInfo := range files {
			writeIndent(out, path, printFiles)

			if idx == len(files)-1 {
				io.WriteString(out, "└───")
			} else {
				io.WriteString(out, "├───")
			}

			io.WriteString(out, fileInfo.Name())
			if !fileInfo.IsDir() {
				if fileInfo.Size() > 0 {
					io.WriteString(out, fmt.Sprintf(" (%db)", fileInfo.Size()))
				} else {
					io.WriteString(out, " (empty)")
				}
			}
			io.WriteString(out, "\n")

			if fileInfo.IsDir() {
				dirTree(out, path+string(os.PathSeparator)+fileInfo.Name(), printFiles)
			}
		}
	} else {
		dirNum := getNumDirsAtCurrLvl(files)
		dirCount := 0
		for _, fileInfo := range files {
			if fileInfo.IsDir() {
				writeIndent(out, path, false)

				if dirCount == dirNum-1 {
					io.WriteString(out, "└───")
				} else {
					io.WriteString(out, "├───")
				}

				io.WriteString(out, fileInfo.Name())
				io.WriteString(out, "\n")

				dirTree(out, path+string(os.PathSeparator)+fileInfo.Name(), printFiles)

				dirCount++
			}
		}
	}
	return nil
}

func writeIndent(out io.Writer, path string, printFiles bool) {
	pathSep := string(os.PathSeparator)
	levels := strings.Split(path, pathSep)
	if len(levels) > 1 {
		path = levels[0]
		for i := 1; i < len(levels); i++ {
			dirs := getFilesInDir(path, printFiles)
			if levels[i] == dirs[len(dirs)-1] {
				io.WriteString(out, "\t")
			} else {
				io.WriteString(out, "│\t")
			}
			path += pathSep + levels[i]
		}
	}
}

func getFilesInDir(path string, printFiles bool) []string {
	file, _ := os.Open(path)
	defer file.Close()

	dirFiles, _ := file.Readdir(-1)
	dirs := make([]string, len(dirFiles))

	for _, dirFile := range dirFiles {
		if printFiles || dirFile.IsDir() {
			dirs = append(dirs, dirFile.Name())
		}
	}

	sort.SliceStable(dirs, func(i, j int) bool {
		return dirs[i] < dirs[j]
	})

	return dirs
}

func getNumDirsAtCurrLvl(fileInfos []os.FileInfo) int {
	c := 0
	for _, info := range fileInfos {
		if info.IsDir() {
			c++
		}
	}
	return c
}
