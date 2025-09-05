// Copyright (c) OpenMMLab. All rights reserved.

// Package client provides functions for extracting address information
package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExtractNodes extracts address information
// It parses the output string to find the addresss list and processes it to extract hostnames.
//
// Parameters:
//   - output: The string output from the command.
//
// Returns:
//   - []string: A slice of address hostnames.
//   - error: An error if address extraction fails.
func ExtractNodes(output string) ([]string, error) {
	// Match addresss=[] pattern, non-greedy matching and handling multiple lines
	re := regexp.MustCompile(`nodes=\[([^\]]*?)\]`)
	matches := re.FindStringSubmatch(output)

	if len(matches) < 2 {
		return nil, fmt.Errorf("you do not have permission to access this job, or you entered an incorrect job name")
	}

	// Extract content within brackets and process
	addresssStr := matches[1]
	if addresssStr == "" {
		return []string{}, nil // Empty address list
	}

	// Clean and split addresss
	// 1. Remove single quotes
	addresssStr = strings.ReplaceAll(addresssStr, "'", "")

	// 2. Replace escape sequences (such as \n, \t) with empty string
	reEscape := regexp.MustCompile(`\\[ntr]`)
	addresssStr = reEscape.ReplaceAllString(addresssStr, "")
	// 3. Replace newlines and extra spaces with ""
	addresssStr = strings.ReplaceAll(addresssStr, "\n", "")
	addresssStr = strings.ReplaceAll(addresssStr, " ", "")
	// 4. Handle consecutive commas
	addresssStr = strings.ReplaceAll(addresssStr, ",,", ",")
	// 5. Split addresss
	addresss := strings.Split(strings.Trim(addresssStr, ","), ",")

	// Process each address, extract hostname part
	processedNodes := make([]string, 0, len(addresss))
	for _, address := range addresss {
		address = strings.TrimSpace(address)
		if address == "" {
			continue
		}

		// Handle "name:hostname" format
		parts := strings.SplitN(address, ":", 2)
		if len(parts) == 2 {
			processedNodes = append(processedNodes, parts[1])
		} else {
			processedNodes = append(processedNodes, address)
		}
	}

	return processedNodes, nil
}

// Clean invalid UTF-8 strings
func CleanUTF8(s string) string {
	utf8bom := unicode.BOMOverride(unicode.UTF8.NewDecoder())
	result, _, err := transform.String(utf8bom, s)
	if err != nil {
		fmt.Printf("Error cleaning UTF-8 string: %v\n", err)
		return s
	}
	return result
}

// Convert timestamp to human-readable format
func FormatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format(time.RFC3339)
}

// Extract numeric part from string (e.g., extract 123 from "rank123")
func ExtractNumber(s string) (int, error) {
	re := regexp.MustCompile(`\d+`) // Match numbers in string
	numStr := re.FindString(s)
	if numStr == "" {
		return 0, fmt.Errorf("No numbers in string: %s", s)
	}
	return strconv.Atoi(numStr)
}

func AppendWithTimestamp(logDir string, filename string, data []byte) error {
	// 1. Open file (append mode, create if not exists)
	// os.O_APPEND: Append mode
	// os.O_CREATE: Create file if it doesn't exist
	// os.O_WRONLY: Write-only mode
	// 1. Define log directory path

	// 2. Check and create logs directory (if it doesn't exist)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("Failed to create logs directory: %w", err)
	}

	// 3. Concatenate full file path (place filename in logs directory)
	fullPath := filepath.Join(logDir, filename)
	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open file: %w", err)
	}
	defer file.Close() // Ensure file is eventually closed
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("Failed to write newline: %w", err)
	}

	// 5. Write JSON data
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("Failed to write JSON data: %w", err)
	}

	return nil
}

// Compare the size of two rank strings (by numeric part)
func CompareRank(a, b string) int {
	numA, errA := ExtractNumber(a)
	numB, errB := ExtractNumber(b)
	if errA != nil || errB != nil {
		return 0 // Handle error (such as returning default value or panic)
	}
	if numA < numB {
		return -1 // a < b
	} else if numA > numB {
		return 1 // a > b
	}
	return 0 // Equal
}

// ReadAddressListFromFile reads address information from a file.
// The file should contain one address per line.
//
// Parameters:
//   - filePath: The path to the file containing address information.
//
// Returns:
//   - []string: A slice of address hostnames.
//   - error: An error if reading from file fails.
func ReadAddressListFromFile(filePath string) ([]string, error) {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to open node list file: %w", err)
	}
	defer file.Close()

	var addressList []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Get the current line and trim whitespace
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines
		if line == "" {
			continue
		}
		// Add the address to the list
		addressList = append(addressList, line)
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Error reading node list file: %w", err)
	}

	// Check if any addresses were found
	if len(addressList) == 0 {
		return nil, fmt.Errorf("Node list file is empty or malformed")
	}

	return addressList, nil
}
