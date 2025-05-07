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
	"sync"
)

// FileEntry represents a single line with a hash and file path.
type FileEntry struct {
	Hash     string
	FilePath string
}

// Result struct to collect output from goroutines
type Result struct {
	IsValid  bool   // Whether the hash matched
	FilePath string // File path
	Message  string // Error or mismatch message, if any
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
		parts := strings.SplitN(line, "  ", 2)

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
		fmt.Println("💥 💥 Please provide the path to the file containing the hash as first argument ")
		os.Exit(1)
	}
	fmt.Printf("ℹ️ Number of arguments received : %d\n", len(args))
	//get the first argument with the file path containing the hash
	hashFilePath := flag.String("hashFilePath", "", "The path to the file containing the hash")
	flag.Parse()
	if *hashFilePath == "" && len(args) == 1 {
		*hashFilePath = args[0]
	}

	fmt.Printf("🏴󠁲󠁯󠁩󠁦󠁿 checking if file exist : %s\n", *hashFilePath)
	if _, err := os.Stat(*hashFilePath); os.IsNotExist(err) {
		fmt.Printf("💥 💥 File not found : %s\n", *hashFilePath)
		os.Exit(1)
	}
	fmt.Printf("✅ Opening file : %s\n", *hashFilePath)
	file, err := os.Open(*hashFilePath)
	if err != nil {
		fmt.Printf("💥 💥 Error opening file : %s\n", err)
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

	// Storing basepath of hash file
	basePath := filepath.Dir(*hashFilePath)
	fmt.Printf("ℹ️ Base path of hash file : %s\n", basePath)

	// Parse the byte slice into a slice of FileEntry structs
	entries, err := parseFileContent(fileContent)
	if err != nil {
		log.Fatalf("Error parsing file content: %v", err)
	}

	// Now 'entries' is a slice of FileEntry structs containing your data
	fmt.Printf("✅ Successfully parsed %d entries.\n", len(entries))
	if len(entries) > 0 {
		var wg sync.WaitGroup
		resultChan := make(chan Result, len(entries)) // Buffered channel to collect results
		numValidHash := 0
		numInvalidHash := 0
		//  Limit concurrency with a worker pool
		const maxWorkers = 20 // Adjust based on system resources
		semaphore := make(chan struct{}, maxWorkers)

		// Process each entry in a goroutine
		for _, entry := range entries {
			wg.Add(1)
			go func(entry FileEntry) {
				defer wg.Done()

				// Acquire semaphore slot (limits concurrent goroutines)
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				//fmt.Printf("Hash: %s, FilePath: %s\n", entry.Hash, entry.FilePath)
				fullPath := filepath.Join(basePath, entry.FilePath)
				hash, err := GetSHA256(fullPath)
				result := Result{}
				if err != nil {
					result.Message = fmt.Sprintf("💥 💥 Error getting hash : %s\n", err)
				} else if hash == entry.Hash {
					result.IsValid = true
				} else {
					result.Message = fmt.Sprintf("❌ ⚠️ 🔥 Hash values do not match for:\t%s\texpecting:\t%s\tgot:\t%s\n", entry.FilePath, entry.Hash, hash)
					result.IsValid = false
					result.FilePath = entry.FilePath
				}
				resultChan <- result
			}(entry)
		}
		// Close result channel after all goroutines finish
		go func() {
			wg.Wait()
			close(resultChan)
		}()

		// Collect results from the channel
		for result := range resultChan {
			if result.Message != "" {
				fmt.Print(result.Message)
			}
			if result.IsValid {
				numValidHash++
			} else {
				numInvalidHash++
			}
		}

		fmt.Printf("✅ File contains %d lines, %d valid hashes and %d invalid hashes.\n", len(entries), numValidHash, numInvalidHash)
	}
}
