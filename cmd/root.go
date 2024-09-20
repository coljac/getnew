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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	sourceDir  string
	nthNewest  int
	fileFilter string
	unarchive  bool
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
		if unarchive {
			if err := unarchiveFile(); err != nil {
				fmt.Fprintf(os.Stderr, "Error unarchiving: %v\n", err)
				os.Exit(1)
			}
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
	rootCmd.Flags().BoolVarP(&unarchive, "unarchive", "z", false, "Unarchive the file if it's an archive (zip, gz, tar.gz, 7z)")

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

	return moveFile(sourceDir, regularFiles, nthNewest, fileFilter)
}

func moveFile(sourceDir string, regularFiles []os.FileInfo, nthNewest int, fileFilter string) error {
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

	// Open the source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy the contents from source to destination
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Close the files
	if err := sourceFile.Close(); err != nil {
		return fmt.Errorf("failed to close source file: %w", err)
	}
	if err := destFile.Close(); err != nil {
		return fmt.Errorf("failed to close destination file: %w", err)
	}

	// Remove the original file
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	fmt.Printf("%s\n", fileToMove.Name())
	return nil
}

func unarchiveFile() error {
	files, err := os.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read current directory: %w", err)
	}

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		var cmd *exec.Cmd

		switch ext {
		case ".zip":
			cmd = exec.Command("unzip", file.Name())
		case ".gz", ".tgz":
			cmd = exec.Command("tar", "-xzf", file.Name())
		case ".tar":
			cmd = exec.Command("tar", "-xf", file.Name())
		case ".7z":
			cmd = exec.Command("7z", "x", file.Name())
		default:
			continue // Not a recognized archive format
		}

		if cmd != nil {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to unarchive %s: %w", file.Name(), err)
			}
			if err := os.Remove(file.Name()); err != nil {
				return fmt.Errorf("failed to remove original archive file: %w", err)
			}
			fmt.Printf("Unarchived and removed: %s\n", file.Name())
			return nil
		}
	}

	return fmt.Errorf("no recognized archive file found in the current directory")
}
