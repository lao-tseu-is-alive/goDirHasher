package hasher

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestGetSHA256 tests the SHA256 hash calculation function.
func TestGetSHA256(t *testing.T) {
	// Create a temporary file with known content
	content := "This is a test file for SHA256 hashing."
	tmpfile, err := os.CreateTemp("", "testfile-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name()) // Clean up the temporary file

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temporary file: %v", err)
	}

	// Calculate the hash of the temporary file
	calculatedHash, err := GetSHA256(tmpfile.Name())
	if err != nil {
		t.Fatalf("GetSHA256 returned an error: %v", err)
	}

	// The expected SHA256 hash for the content "This is a test file for SHA256 hashing."
	// You can verify this using a command-line tool like `echo -n "This is a test file for SHA256 hashing." | sha256sum`
	expectedHash := "9C495B60E232739F0E1777969172C84520F33837C877A6719A9256676F26927F"

	// Compare the calculated hash with the expected hash
	if calculatedHash != expectedHash {
		t.Errorf("GetSHA256(%q) returned %q, expected %q", tmpfile.Name(), calculatedHash, expectedHash)
	}

	// Test with a non-existent file
	_, err = GetSHA256("non_existent_file.txt")
	if err == nil {
		t.Error("GetSHA256 for non-existent file did not return an error")
	}
}

// TestParseHashFile tests the function that parses the hash file content.
func TestParseHashFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []FileEntry
		wantErr  bool
	}{
		{
			name: "Valid input",
			input: `
ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789  file1.txt
FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210  path/to/file2.dat
`,
			expected: []FileEntry{
				{Hash: "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789", FilePath: "file1.txt"},
				{Hash: "FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210", FilePath: "path/to/file2.dat"},
			},
			wantErr: false,
		},
		{
			name: "Input with empty lines and comments",
			input: `
# This is a comment

ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789  file1.txt

# Another comment


  FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210  file2.txt

`,
			expected: []FileEntry{
				{Hash: "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789", FilePath: "file1.txt"},
				{Hash: "FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210", FilePath: "file2.txt"},
			},
			wantErr: false,
		},
		{
			name: "Input with incorrect format (missing spaces)",
			input: `
ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789 file1.txt
FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210  file2.txt
`,
			expected: []FileEntry{
				// The first line should be skipped due to incorrect format
				{Hash: "FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210FEDCBA9876543210", FilePath: "file2.txt"},
			},
			wantErr: false, // We expect it to log a warning but not return an error for a single bad line
		},
		{
			name:     "Empty input",
			input:    "",
			expected: []FileEntry{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a strings.Reader to simulate reading from a file
			reader := strings.NewReader(tt.input)
			entries, err := ParseHashFile(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHashFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Compare the parsed entries with the expected ones
			if len(entries) != len(tt.expected) {
				t.Errorf("ParseHashFile() returned %d entries, expected %d", len(entries), len(tt.expected))
				return
			}

			for i := range entries {
				if entries[i].Hash != tt.expected[i].Hash {
					t.Errorf("Entry %d: Hash mismatch. Got %q, expected %q", i, entries[i].Hash, tt.expected[i].Hash)
				}
				if entries[i].FilePath != tt.expected[i].FilePath {
					t.Errorf("Entry %d: FilePath mismatch. Got %q, expected %q", i, entries[i].FilePath, tt.expected[i].FilePath)
				}
			}
		})
	}
}

// Helper function to create a dummy file for testing
func createDummyFile(t *testing.T, name string, content string) string {
	tmpfile, err := os.CreateTemp("", name)
	if err != nil {
		t.Fatalf("Failed to create temporary file %q: %v", name, err)
	}
	defer tmpfile.Close() // Close the file handle immediately

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temporary file %q: %v", name, err)
	}

	return tmpfile.Name() // Return the full path to the temporary file
}

// Test integration between GetSHA256 and ParseHashFile (basic check)
func TestGetSHA256AndParseHashFileIntegration(t *testing.T) {
	// Create a dummy file
	fileContent := "Content for integration test."
	dummyFilePath := createDummyFile(t, "integration-test-*.txt", fileContent)
	defer os.Remove(dummyFilePath) // Clean up

	// Calculate its hash
	calculatedHash, err := GetSHA256(dummyFilePath)
	if err != nil {
		t.Fatalf("GetSHA256 failed: %v", err)
	}

	// Create a simulated hash file content using the calculated hash and dummy file path
	hashFileContent := fmt.Sprintf("%s  %s\n", calculatedHash, dummyFilePath)
	reader := strings.NewReader(hashFileContent)

	// Parse the simulated hash file content
	entries, err := ParseHashFile(reader)
	if err != nil {
		t.Fatalf("ParseHashFile failed: %v", err)
	}

	// Verify the parsed entry
	if len(entries) != 1 {
		t.Fatalf("Expected 1 parsed entry, got %d", len(entries))
	}

	parsedEntry := entries[0]

	if parsedEntry.Hash != calculatedHash {
		t.Errorf("Parsed hash %q does not match calculated hash %q", parsedEntry.Hash, calculatedHash)
	}

	if parsedEntry.FilePath != dummyFilePath {
		// Note: Depending on how you want to handle paths in your hash file vs actual file paths,
		// this comparison might need adjustment if you expect relative paths in the hash file.
		// For this basic test, we're using the full temporary file path.
		t.Errorf("Parsed file path %q does not match dummy file path %q", parsedEntry.FilePath, dummyFilePath)
	}
}
