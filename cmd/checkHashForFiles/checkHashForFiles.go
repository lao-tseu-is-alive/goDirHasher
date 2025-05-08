package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"flag"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const defaultMaxWorkers = 15

// FileEntry represents a single line with a hash and file path.
type FileEntry struct {
	Hash     string
	FilePath string
}

// Result struct to collect output from goroutines
type Result struct {
	IsValid bool   // Whether the hash matched
	Message string // Error or mismatch message, if any
}

// parseFileContent converts a byte slice into a slice of FileEntry structs.
// Each line is expected to be in the format "hash filepath".
func parseFileContent(file *os.File) ([]FileEntry, error) {
	var entries []FileEntry
	scanner := bufio.NewScanner(file)

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
			log.Printf("Warning: Skipping line %d due to incorrect format: %s\n", lineNumber, line)
			continue
		}
		// Create a FileEntry struct and append it to the slice
		entries = append(entries, FileEntry{
			Hash:     strings.ToUpper(parts[0]),
			FilePath: parts[1],
		})
	}
	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading lines: %w", err)
	}
	return entries, nil
}

// sha256HashPool holds reusable SHA-256 hash instances.
var sha256HashPool = sync.Pool{
	New: func() any {
		// On pool miss, allocate a fresh hasher.
		return sha256.New()
	},
}

// Buffer pool for I/O operations
var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 64*1024) // 64KB buffer, adjustable
	},
}

// GetMD5 returns md5 hash of a file
func GetMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", h.Sum(nil)), nil
}

// GetSHA256 returns sha256 hash of a file
func GetSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// Retrieve a hasher from the pool (or New() if empty)
	h := sha256HashPool.Get().(hash.Hash)
	// Reset its internal state before reuse
	h.Reset()
	// Return it to the pool when done
	defer sha256HashPool.Put(h)

	// Wrap in a buffered reader to reduce syscalls
	br := bufio.NewReader(f)
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)
	if _, err := io.CopyBuffer(h, br, buf); err != nil {
		return "", err
	}
	// Compute checksum
	sum := h.Sum(nil)
	return fmt.Sprintf("%X", sum), nil
}

func displayUsageAndExit() {
	fmt.Printf("💥 💥 Missing arguments, Usage : %s\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Println("💥 💥 Please provide at least the path to the file containing the hash as first argument ")
	os.Exit(1)
}

// this will generate hash for all files in the given directory and compare
func main() {
	//get the first argument with the file path containing the hash
	hashFilePath := flag.String("hashFilePath", "", "The path to the file containing the hash")
	var maxWorkers int
	flag.IntVar(&maxWorkers, "workers", defaultMaxWorkers, "Number of concurrent workers")
	flag.Parse()
	args := os.Args[1:]
	if len(args) == 0 {
		displayUsageAndExit()
	}
	if len(args) < 1 {
		fmt.Println("💥 💥 Please provide at least the path to the file containing the hash as first argument ")
		flag.PrintDefaults()
		os.Exit(1)
	}
	fmt.Printf("ℹ️ Number of arguments received : %d\n", len(args))
	// Ensure maxWorkers is reasonable
	if maxWorkers < 1 {
		maxWorkers = defaultMaxWorkers
	}
	if maxWorkers > 50 {
		maxWorkers = 50
	}
	if *hashFilePath == "" && len(args) == 1 {
		*hashFilePath = args[0]
	} else if *hashFilePath == "" && len(args) > 1 {
		displayUsageAndExit()
	}
	fmt.Printf("ℹ️ Using maxWorkers = %d \n", maxWorkers)
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
	// Storing basepath of the file
	basePath := filepath.Dir(*hashFilePath)
	fmt.Printf("ℹ️ Base path of hash file : %s\n", basePath)
	// Parse the byte slice into a slice of FileEntry structs
	entries, err := parseFileContent(file)
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
		semaphore := make(chan struct{}, maxWorkers)

		// Process each entry in a goroutine
		for _, entry := range entries {
			wg.Add(1)
			go func(entry FileEntry) {
				defer wg.Done()
				// Acquire semaphore slot (limits concurrent goroutines)
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				fullPath := filepath.Join(basePath, entry.FilePath)
				fileHash, err := GetSHA256(fullPath)
				//fileHash, err := GetMD5(fullPath)
				result := Result{}
				if err != nil {
					result.Message = fmt.Sprintf("💥 💥 Error getting hash : %s\n", err)
				} else if fileHash == entry.Hash {
					result.IsValid = true
				} else {
					result.Message = fmt.Sprintf("❌ ⚠️ 🔥 Hash values do not match for:\t%s\texpecting:\t%s\tgot:\t%s\n", entry.FilePath, entry.Hash, fileHash)
				}
				resultChan <- result
			}(entry)
		}
		// Close the result channel after all goroutines finish
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
