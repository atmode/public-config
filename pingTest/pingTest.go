package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	pingTestURL   = "google.com" // URL to test connectivity
	pingTimeout   = 5 * time.Second
	xrayPath      = "xray"        // Path to xray executable
	tempConfigDir = "temp_config" // Directory to store temporary config files
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <config_file.txt>")
		return
	}

	configFilePath := os.Args[1]

	// Create temporary directory for config files
	err := os.MkdirAll(tempConfigDir, 0755)
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempConfigDir)

	// Read the configs file
	file, err := os.Open(configFilePath)
	if err != nil {
		fmt.Printf("Error opening config file: %v\n", err)
		return
	}
	defer file.Close()

	// Prepare the output file
	outputPath := filepath.Join(
		filepath.Dir(configFilePath),
		"working_"+filepath.Base(configFilePath),
	)
	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(outputFile)

	lineNum := 0
	workingConfigs := 0

	fmt.Println("Starting Xray config ping test...")
	fmt.Println("==================================")

	for scanner.Scan() {
		lineNum++
		configLine := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(configLine) == "" {
			continue
		}

		fmt.Printf("[%d] Testing config: %s\n", lineNum, truncateString(configLine, 50))

		// Test the config
		pingTime, err := testXrayConfig(configLine)

		if err != nil {
			fmt.Printf("    ❌ Failed: %v\n", err)
			continue
		}

		// If we got here, the config works!
		fmt.Printf("    ✅ Working! Ping: %v ms\n", pingTime)

		// Write the working config to the output file
		fmt.Fprintf(writer, "%s\n", configLine)
		workingConfigs++
	}

	// Flush the writer
	writer.Flush()

	fmt.Println("==================================")
	fmt.Printf("Testing complete: %d configs tested, %d working configs\n", lineNum, workingConfigs)
	fmt.Printf("Working configs saved to: %s\n", outputPath)
}

// testXrayConfig tests if an Xray config works and returns the ping time
func testXrayConfig(configLine string) (int, error) {
	// Create a temporary config file
	configFile, err := createTempConfig(configLine)
	if err != nil {
		return 0, fmt.Errorf("failed to create config: %v", err)
	}
	defer os.Remove(configFile)

	// Start Xray with the config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, xrayPath, "-c", configFile)

	// Redirect stderr to capture any error messages
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	// Start Xray
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start Xray: %v", err)
	}

	// Give Xray some time to initialize
	time.Sleep(1 * time.Second)

	// Check if Xray started successfully
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		// Read error output
		errorOutput, _ := io.ReadAll(stderr)
		return 0, fmt.Errorf("Xray exited early: %s", string(errorOutput))
	}

	// Test the connection using curl with the SOCKS proxy
	// Adjust port if your Xray config uses a different port
	pingCmd := exec.Command("curl", "--socks5", "127.0.0.1:1080",
		"--connect-timeout", strconv.Itoa(int(pingTimeout.Seconds())),
		"-o", "/dev/null", "-s", "-w", "%{time_total}", pingTestURL)

	output, err := pingCmd.Output()
	if err != nil {
		cancel() // Stop Xray
		return 0, fmt.Errorf("connection test failed: %v", err)
	}

	// Parse the ping time
	pingTimeStr := strings.TrimSpace(string(output))
	pingTimeFloat, err := strconv.ParseFloat(pingTimeStr, 64)
	if err != nil {
		cancel() // Stop Xray
		return 0, fmt.Errorf("failed to parse ping time: %v", err)
	}

	// Convert to milliseconds
	pingTimeMs := int(pingTimeFloat * 1000)

	// Stop Xray
	cancel()

	return pingTimeMs, nil
}

// createTempConfig creates a temporary config file for Xray
func createTempConfig(configLine string) (string, error) {
	// Parse the config line and create a proper Xray config
	// This is a simplified example. You may need to adjust based on your config format.

	// For this example, assuming configLine is a complete Xray config in JSON format
	configFilePath := filepath.Join(tempConfigDir, fmt.Sprintf("config_%d.json", time.Now().UnixNano()))

	err := os.WriteFile(configFilePath, []byte(configLine), 0644)
	if err != nil {
		return "", err
	}

	return configFilePath, nil
}

// Helper function to truncate long strings for display
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
