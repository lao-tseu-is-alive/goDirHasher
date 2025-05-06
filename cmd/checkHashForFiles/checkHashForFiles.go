package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// FileEntry represents a single line with a hash and file path.
type FileEntry struct {
	Hash     string
	FilePath string
}

// parseFileContent converts a byte slice into a slice of FileEntry structs.
// Each line is expected to be in the format "hash filepath".
func parseFileContent(content []byte) ([]FileEntry, error) {
	var entries []FileEntry
	byteReader := bytes.NewReader(content)
	scanner := bufio.NewScanner(byteReader)

	// Set the scanner to split by lines
	scanner.Split(bufio.ScanLines)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Skip empty lines
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// Split the line into hash and file path by the first space
		parts := strings.SplitN(line, " ", 2)

		if len(parts) != 2 {
			// Log a warning or return an error for lines that don't match the format
			log.Printf("Warning: Skipping line %d due to incorrect format: %s\n", lineNumber, line)
			continue
			// Or, if you want to treat malformed lines as a fatal error:
			// return nil, fmt.Errorf("line %d has incorrect format: %s", lineNumber, line)
		}

		hash := parts[0]
		filePath := parts[1]

		// Create a FileEntry struct and append it to the slice
		entries = append(entries, FileEntry{
			Hash:     hash,
			FilePath: filePath,
		})
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading lines: %w", err)
	}

	return entries, nil
}

// GetSHA256 returns sha256 hash of a file
func GetSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("💥 💥 File not found : %v\n", err)
		}
	}(f)
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// this will generate hash for all files in the given directory and compare
func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("💥 💥 Please provide the path to the directory containing the files hash as first argument ")
		return
	}
	fmt.Printf("✅ Number of arguments received : %d\n", len(args))
	//get the first argument with the file path containing the hash
	hashFilePath := flag.String("hashFilePath", "", "The path to the file containing the hash")
	flag.Parse()
	if *hashFilePath == "" && len(args) == 1 {
		*hashFilePath = args[0]
	}

	fmt.Printf("✅ checking if file exist : %s\n", *hashFilePath)
	if _, err := os.Stat(*hashFilePath); os.IsNotExist(err) {
		fmt.Printf("💥 💥 File not found : %s\n", *hashFilePath)
		os.Exit(1)
	}
	fmt.Printf("✅ Reading file : %s\n", *hashFilePath)
	file, err := os.Open(*hashFilePath)
	if err != nil {
		fmt.Printf("💥 💥 Error reading file : %s\n", err)
		os.Exit(1)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("💥 💥 Error closing file : %s\n", err)
		}
	}(file)
	fileContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("💥 💥 Error reading file : %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ File read successfully : %s\n", *hashFilePath)

	// storing basepath of hash
	basePath := filepath.Dir(*hashFilePath)
	fmt.Printf("✅ Base path of hash file : %s\n", basePath)

	// Parse the byte slice into a slice of FileEntry structs
	entries, err := parseFileContent(fileContent)
	if err != nil {
		log.Fatalf("Error parsing file content: %v", err)
	}

	// Now 'entries' is a slice of FileEntry structs containing your data
	fmt.Printf("✅ Successfully parsed %d entries.\n", len(entries))

	// Example of accessing the parsed data:
	if len(entries) > 0 {
		fmt.Printf("First entry - Hash: %s, FilePath: %s\n", entries[0].Hash, entries[0].FilePath)
		fullPath := filepath.Join(basePath, strings.TrimSpace(entries[0].FilePath))
		hash, err := GetSHA256(fullPath)
		if err != nil {
			fmt.Printf("💥 💥 Error getting hash : %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Successfully got hash : %s\n", hash)
		// comparing hash values
		if hash == entries[0].Hash {
			fmt.Printf("✅ Hash values match\n")
		} else {
			fmt.Printf("❎⚠️ Hash values do not match\n")
		}
	}

	fmt.Printf("✅ File contains %d lines\n", len(entries))

}
