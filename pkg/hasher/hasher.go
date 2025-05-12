package hasher

import (
	"bufio"
	"crypto/md5" // Keeping MD5 for now, but focus is on SHA256
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// FileEntry represents a single line with a hash and file path.
type FileEntry struct {
	Hash     string
	FilePath string
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
// Kept for potential future use or comparison, but SHA256 is preferred.
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

// GetSHA256 returns sha256 hash of a file at the given path.
// It uses a sync.Pool for hashers and a buffer pool for efficiency.
func GetSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Retrieve a hasher from the pool (or New() if empty)
	shaWriter := sha256HashPool.Get().(hash.Hash)
	// Reset its internal state before reuse
	shaWriter.Reset()
	// Return it to the pool when done
	defer sha256HashPool.Put(shaWriter)

	// Wrap in a buffered reader to reduce syscalls
	br := bufio.NewReader(f)
	// Get buffer from pool
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf) // Return buffer to pool

	// Copy file content to the hasher
	if _, err := io.CopyBuffer(shaWriter, br, buf); err != nil {
		return "", err
	}

	// Calculate the final hash sum
	sum := shaWriter.Sum(nil)
	// Format the hash as a hexadecimal string
	return fmt.Sprintf("%X", sum), nil
}

// ParseHashFile reads a file line by line, expecting each line to be in
// the format "hash filepath". It returns a slice of FileEntry structs.
// It takes an io.Reader for flexibility (can read from file, stdin, etc.).
func ParseHashFile(reader io.Reader) ([]FileEntry, error) {
	var entries []FileEntry
	scanner := bufio.NewScanner(reader)

	// Set the scanner to split by lines
	scanner.Split(bufio.ScanLines)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		// Skip empty lines and lines starting with # (comments)
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// Split the line into hash and file path by the first two spaces (standard sha256sum format)
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			// Log a warning and skip lines that don't match the expected format
			log.Printf("Warning: Skipping line %d due to incorrect format: %s\n", lineNumber, line)
			continue
		}

		// Create a FileEntry struct and append it to the slice
		// Trim spaces from hash and filepath parts
		entries = append(entries, FileEntry{
			Hash:     strings.ToUpper(strings.TrimSpace(parts[0])), // Ensure hash is uppercase
			FilePath: strings.TrimSpace(parts[1]),
		})
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading lines: %w", err)
	}

	return entries, nil
}
