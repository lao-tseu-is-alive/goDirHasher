package main

import (
	"flag"
	"fmt"
	"github.com/lao-tseu-is-alive/goDirHasher/pkg/hasher"
	"github.com/lao-tseu-is-alive/goDirHasher/pkg/version"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
)

const defaultMaxWorkers = 15

// CheckResult Result struct to collect output from goroutines during checking
type CheckResult struct {
	FilePath string // The file path being checked
	IsValid  bool   // Whether the hash matched
	Message  string // Error or mismatch message, if any
}

// CalcResult Result struct to collect output from goroutines during calculation
type CalcResult struct {
	FilePath string // The file path being processed
	Hash     string // The calculated hash
	Error    error  // Any error encountered
}

// displayUsageAndExit prints the command usage and exits.
func displayUsageAndExit() {
	fmt.Printf("Usage: %s [OPTIONS] [FILE...]\n", os.Args[0])
	fmt.Println("\nCalculates or checks SHA256 hashes of files.")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
	fmt.Println("\nArguments:")
	fmt.Println("  FILE...    Files or directories to process.")
	fmt.Println("             If no files are specified, reads from standard input.")
	fmt.Println("             In check mode (-c), FILE is the hash file to read.")
	fmt.Println("\nExamples:")
	fmt.Println("  Calculate hash for a file: go run main.go myfile.txt")
	fmt.Println("  Calculate hashes for multiple files: go run main.go file1.txt dir1/file2.txt")
	fmt.Println("  Calculate hashes for all files in current directory: go run main.go .")
	fmt.Println("  Calculate hashes and save to file: go run main.go . > hashes.txt")
	fmt.Println("  Check hashes from a file: go run main.go -c hashes.txt")
	fmt.Println("  Check hashes from stdin: cat hashes.txt | go run main.go -c -") // Use '-' for stdin
	os.Exit(1)
}

func main() {
	fmt.Printf("🚀 Starting App:'%s', ver:%s, BuildStamp: %s, Repo: %s\n", version.APP, version.VERSION, version.BuildStamp, version.REPOSITORY)

	// Command-line flags
	checkMode := flag.Bool("c", false, "Check hashes against a file (or stdin)")
	outputFile := flag.String("o", "", "Output file for calculated hashes (defaults to stdout)")
	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file")
	memProfile := flag.String("memprofile", "", "Write memory profile to file")
	var maxWorkers int
	flag.IntVar(&maxWorkers, "workers", defaultMaxWorkers, "Number of concurrent workers")
	flag.Parse()

	// Start CPU profiling if requested
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatalf("Could not create CPU profile: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("Could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	// Ensure maxWorkers is reasonable
	if maxWorkers < 1 {
		maxWorkers = defaultMaxWorkers
	}
	if maxWorkers > 50 { // Cap workers to avoid overwhelming the system
		maxWorkers = 50
	}
	fmt.Printf("ℹ️ Using maxWorkers = %d \n", maxWorkers)

	// Get the list of files/directories to process from arguments
	args := flag.Args()

	// Determine the mode (calculate or check) and process accordingly
	if *checkMode {
		// --- Check Mode ---
		fmt.Println("🕵️ Entering check mode...")

		var hashFileReader io.Reader
		hashFilePath := ""

		if len(args) == 0 {
			// No file specified, read from stdin
			fmt.Println("ℹ️ Reading hash data from standard input...")
			hashFileReader = os.Stdin
			hashFilePath = "stdin" // Just for logging/messages
		} else if len(args) == 1 {
			// Read from the specified hash file
			hashFilePath = args[0]
			fmt.Printf("🏴󠁲󠁯󠁩󠁦󠁿 Checking if hash file exists: %s\n", hashFilePath)
			file, err := os.Open(hashFilePath)
			if err != nil {
				log.Fatalf("💥 💥 Error opening hash file %s: %v", hashFilePath, err)
			}
			defer file.Close()
			hashFileReader = file
			fmt.Printf("✅ Opening hash file: %s\n", hashFilePath)
		} else {
			// Too many arguments in check mode
			fmt.Println("💥 💥 In check mode (-c), provide at most one argument (the hash file path or '-' for stdin).")
			displayUsageAndExit()
		}

		// Parse the hash file content
		entries, err := hasher.ParseHashFile(hashFileReader)
		if err != nil {
			log.Fatalf("Error parsing hash file %s: %v", hashFilePath, err)
		}

		fmt.Printf("✅ Successfully parsed %d entries from %s.\n", len(entries), hashFilePath)

		if len(entries) == 0 {
			fmt.Println("ℹ️ No hash entries found in the file. Nothing to check.")
			os.Exit(0)
		}

		var wg sync.WaitGroup
		checkResultChan := make(chan CheckResult, len(entries)) // Buffered channel to collect results
		semaphore := make(chan struct{}, maxWorkers)            //  Limit concurrency with a worker pool

		// Process each entry in a goroutine
		for _, entry := range entries {
			wg.Add(1)
			go func(entry hasher.FileEntry) {
				defer wg.Done()
				// Acquire semaphore slot (limits concurrent goroutines)
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Determine the full path relative to the hash file's directory
				// If reading from stdin, assume paths are relative to the current directory
				basePath := filepath.Dir(hashFilePath)
				if hashFilePath == "stdin" || basePath == "." {
					basePath = "." // Use current directory if reading from stdin or file is in current dir
				} else {
					// If hash file is in a subdirectory, join paths
					// Need to handle cases where entry.FilePath is absolute vs relative
					// For simplicity here, assuming relative paths in hash file are relative to hash file dir
					// A more robust solution might involve a --directory flag
					// For now, let's assume paths in the hash file are relative to the hash file's location
					// unless they are absolute paths.
					if !filepath.IsAbs(entry.FilePath) {
						basePath = filepath.Dir(hashFilePath)
					} else {
						basePath = "" // If absolute path, no base path needed
					}
				}

				fullPath := filepath.Join(basePath, entry.FilePath)
				// Clean the path to handle cases like "./file.txt"
				fullPath = filepath.Clean(fullPath)

				fileHash, err := hasher.GetSHA256(fullPath)
				result := CheckResult{FilePath: entry.FilePath} // Use original path from file for reporting

				if err != nil {
					result.Message = fmt.Sprintf("💥 💥 Error getting hash for %s: %v\n", entry.FilePath, err)
					result.IsValid = false // Treat error as invalid
				} else if strings.ToUpper(fileHash) == entry.Hash { // Compare uppercase hashes
					result.IsValid = true
					// Optional: Print success messages, but sha256sum usually only prints failures
					// fmt.Printf("✅ %s: OK\n", entry.FilePath)
				} else {
					result.Message = fmt.Sprintf("❌ ⚠️ 🔥 %s: FAILED\n", entry.FilePath)
					// Optional: Print expected vs got hash on failure
					// result.Message += fmt.Sprintf("    Expected: %s\n    Got:      %s\n", entry.Hash, fileHash)
					result.IsValid = false
				}
				checkResultChan <- result
			}(entry)
		}

		// Close the result channel after all goroutines finish
		go func() {
			wg.Wait()
			close(checkResultChan)
		}()

		// Collect results from the channel
		numValidHash := 0
		numInvalidHash := 0
		hasFailure := false

		for result := range checkResultChan {
			if result.Message != "" {
				fmt.Print(result.Message)
			}
			if result.IsValid {
				numValidHash++
			} else {
				numInvalidHash++
				hasFailure = true
			}
		}

		if numInvalidHash > 0 {
			fmt.Printf("⚠️ WARNING: %d computed hash%s did not match\n", numInvalidHash, func() string {
				if numInvalidHash > 1 {
					return "es"
				} else {
					return ""
				}
			}())
		}
		fmt.Printf("✅ %d file%s processed, %d valid, %d invalid.\n", len(entries), func() string {
			if len(entries) > 1 {
				return "s"
			} else {
				return ""
			}
		}(), numValidHash, numInvalidHash)

		if hasFailure {
			os.Exit(1) // Exit with non-zero status on failure
		}

	} else {
		// --- Calculate Mode ---
		fmt.Println("🔢 Entering calculate mode...")

		if len(args) == 0 {
			fmt.Println("💥 💥 No files or directories specified for calculation.")
			displayUsageAndExit()
		}

		var filesToProcess []string

		// Walk directories and add files to the list
		for _, arg := range args {
			info, err := os.Stat(arg)
			if err != nil {
				log.Printf("💥 💥 Error stating %s: %v. Skipping.\n", arg, err)
				continue
			}

			if info.IsDir() {
				// Walk the directory and add files
				err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						log.Printf("💥 💥 Error accessing path %s: %v. Skipping.\n", path, err)
						return nil // Don't stop the walk, just skip this file/dir
					}
					if !info.IsDir() {
						filesToProcess = append(filesToProcess, path)
					}
					return nil
				})
				if err != nil {
					log.Fatalf("💥 💥 Error walking directory %s: %v", arg, err)
				}
			} else {
				// Add the single file
				filesToProcess = append(filesToProcess, arg)
			}
		}

		if len(filesToProcess) == 0 {
			fmt.Println("ℹ️ No files found to calculate hashes for.")
			os.Exit(0)
		}

		fmt.Printf("ℹ️ Found %d file%s to process.\n", len(filesToProcess), func() string {
			if len(filesToProcess) > 1 {
				return "s"
			} else {
				return ""
			}
		}())

		var wg sync.WaitGroup
		calcResultChan := make(chan CalcResult, len(filesToProcess)) // Buffered channel for results
		semaphore := make(chan struct{}, maxWorkers)                 // Limit concurrency

		// Process each file in a goroutine
		for _, filePath := range filesToProcess {
			wg.Add(1)
			go func(filePath string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				hash, err := hasher.GetSHA256(filePath)
				calcResultChan <- CalcResult{FilePath: filePath, Hash: hash, Error: err}
			}(filePath)
		}

		// Close the result channel after all goroutines finish
		go func() {
			wg.Wait()
			close(calcResultChan)
		}()

		// Determine output writer
		var outputWriter io.Writer = os.Stdout
		var outFile *os.File
		if *outputFile != "" {
			var err error
			outFile, err = os.Create(*outputFile)
			if err != nil {
				log.Fatalf("💥 💥 Error creating output file %s: %v", *outputFile, err)
			}
			defer outFile.Close()
			outputWriter = outFile
			fmt.Printf("ℹ️ Writing output to file: %s\n", *outputFile)
		} else {
			fmt.Println("ℹ️ Writing output to standard output.")
		}

		// Collect results and write to output
		errorCount := 0
		for result := range calcResultChan {
			if result.Error != nil {
				log.Printf("💥 💥 Error calculating hash for %s: %v", result.FilePath, result.Error)
				errorCount++
			} else {
				// sha256sum format: hash  filepath
				// Use relative path if possible, or absolute path if needed.
				// For simplicity, let's output the path as provided or found by walk
				// A more sophisticated version might calculate relative paths from a base directory.
				fmt.Fprintf(outputWriter, "%s  %s\n", result.Hash, result.FilePath)
			}
		}

		if errorCount > 0 {
			fmt.Printf("⚠️ WARNING: Encountered %d error%s during hash calculation.\n", errorCount, func() string {
				if errorCount > 1 {
					return "s"
				} else {
					return ""
				}
			}())
			os.Exit(1) // Exit with non-zero status on errors
		} else {
			fmt.Printf("✅ Successfully calculated hashes for %d file%s.\n", len(filesToProcess), func() string {
				if len(filesToProcess) > 1 {
					return "s"
				} else {
					return ""
				}
			}())
		}
	}

	// Write a memory profile if requested
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatalf("Could not create memory profile: %v", err)
		}
		defer f.Close()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatalf("Could not write memory profile: %v", err)
		}
	}
}
