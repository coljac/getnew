/*
Copyright © 2024 Colin Jacobs <colin@coljac.space>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	sourceDir string
	nthNewest int
	fileFilter string
)

var rootCmd = &cobra.Command{
	Use:   "getnew [filter]",
	Short: "Move the nth newest file from a source directory to the current directory",
	Long: `getnew is a CLI tool that looks in a specified directory for the nth newest file
and moves it to the current directory. By default, it moves the newest file.

The source directory can be set using the GETNEW_SOURCE_DIR environment variable
or specified using the --source flag.

Optionally, provide a filter argument to match files partially.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fileFilter = args[0]
		}
		if err := moveNthNewestFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&sourceDir, "source", "s", "", "Source directory (overrides GETNEW_SOURCE_DIR)")
	rootCmd.Flags().IntVarP(&nthNewest, "nth", "n", 1, "Nth newest file to move (default is 1, the newest)")

	// Use environment variable if --source flag is not set
	if sourceDir == "" {
		sourceDir = os.Getenv("GETNEW_SOURCE_DIR")
		if sourceDir == "" {
			sourceDir = filepath.Join(os.Getenv("HOME"), "Downloads") // Default to ~/Downloads if not set
		}
	}
}

func moveNthNewestFile() error {
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	var regularFiles []os.FileInfo
	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err != nil {
				return fmt.Errorf("failed to get file info: %w", err)
			}
			if fileFilter == "" || strings.Contains(strings.ToLower(info.Name()), strings.ToLower(fileFilter)) {
				regularFiles = append(regularFiles, info)
			}
		}
	}

	if len(regularFiles) == 0 {
		if fileFilter != "" {
			return fmt.Errorf("no files matching '%s' found in the source directory", fileFilter)
		}
		return fmt.Errorf("no files found in the source directory")
	}

	sort.Slice(regularFiles, func(i, j int) bool {
		return regularFiles[i].ModTime().After(regularFiles[j].ModTime())
	})

	if nthNewest > len(regularFiles) {
		return fmt.Errorf("requested %dth newest file, but only %d files available", nthNewest, len(regularFiles))
	}

	fileToMove := regularFiles[nthNewest-1]
	sourcePath := filepath.Join(sourceDir, fileToMove.Name())
	destPath := filepath.Join(".", fileToMove.Name())

	if err := os.Rename(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	fmt.Printf("Moved %s to current directory\n", fileToMove.Name())
	return nil
}
