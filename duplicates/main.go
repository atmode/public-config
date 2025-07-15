package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <filename>")
		return
	}

	filename := os.Args[1]

	absPath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		return
	}

	fmt.Printf("Opening: %s\n", absPath)

	file, err := os.Open(absPath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	findDuplicates(lines)

	removeDuplicates(lines, absPath)
}

func findDuplicates(lines []string) {
	processed := make(map[string]bool)

	for _, line := range lines {
		if processed[line] {
			continue
		}

		processed[line] = true

		count := 0
		for _, otherLine := range lines {
			if line == otherLine {
				count++
			}
		}

		if count > 1 {
			fmt.Printf("Duplicate found: \"%s\" - %d occurrences\n", line, count)
		}
	}
}

func removeDuplicates(lines []string, originalPath string) {
	uniqueLines := make(map[string]bool)
	var result []string

	for _, line := range lines {
		if !uniqueLines[line] {
			uniqueLines[line] = true
			result = append(result, line)
		}
	}

	dir := filepath.Dir(originalPath)
	baseName := filepath.Base(originalPath)
	newFilePath := filepath.Join(dir, "removedDuplicate_"+baseName)

	outFile, err := os.Create(newFilePath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	for _, line := range result {
		fmt.Fprintln(writer, line)
	}

	writer.Flush()

	fmt.Printf("Duplicates removed! New file created: %s\n", newFilePath)
	fmt.Printf("Original line count: %d, New line count: %d\n", len(lines), len(result))
}
