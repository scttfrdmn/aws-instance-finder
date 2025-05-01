// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright 2025, Scott Friedman and Project Contributors

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// Version information
const (
	version    = "0.9.0"
	dateFormat = "2006-01"
)

func main() {
	// Parse command line arguments
	versionFlag := flag.Bool("version", false, "Show version information")
	startMonth := flag.String("start", "", "Start month (YYYY-MM)")
	endMonth := flag.String("end", "", "End month (YYYY-MM)")
	instanceTypesStr := flag.String("instances", "", "Comma-separated list of instance types (use type. for all of that type, e.g., t3.)")
	showCost := flag.Bool("show-cost", false, "Show cost data")
	showUsage := flag.Bool("show-usage", false, "Show usage data")
	outputFile := flag.String("output", "", "Output file path (CSV)")
	useSso := flag.Bool("sso", false, "Use AWS SSO authentication (requires AWS CLI)")
	profileName := flag.String("profile", "", "AWS profile name to use")

	flag.Parse()

	// Show version if requested
	if *versionFlag {
		fmt.Printf("AWS Instance Finder v%s\n", version)
		fmt.Printf("  - Go version: %s\n", runtime.Version())
		fmt.Printf("  - OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// Validate required arguments
	if *startMonth == "" || *endMonth == "" || *instanceTypesStr == "" {
		fmt.Println("Error: start, end, and instances are required parameters")
		flag.Usage()
		os.Exit(1)
	}

	// Process instance types
	instanceTypes := strings.Split(*instanceTypesStr, ",")
	for i, inst := range instanceTypes {
		instanceTypes[i] = strings.TrimSpace(inst)
	}

	// Set up AWS config with appropriate authentication
	var cfg config.Config
	var err error

	if *useSso {
		fmt.Println("Using AWS SSO for authentication...")
		// Check if SSO session is active, if not, initiate login
		if err := ensureSsoLogin(*profileName); err != nil {
			log.Fatalf("Error with SSO login: %v", err)
		}

		// Load config with SSO credentials
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithSharedConfigProfile(*profileName),
		)
	} else {
		// Use default credential chain with optional profile
		cfgOptions := []func(*config.LoadOptions) error{}
		if *profileName != "" {
			cfgOptions = append(cfgOptions, config.WithSharedConfigProfile(*profileName))
		}
		cfg, err = config.LoadDefaultConfig(context.TODO(), cfgOptions...)
	}

	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	// Create Cost Explorer client
	ce := costexplorer.NewFromConfig(cfg)

	// Get instance usage
	result, err := getInstanceUsage(ce, *startMonth, *endMonth, instanceTypes, *showCost, *showUsage)
	if err != nil {
		log.Fatalf("Error getting instance usage: %v", err)
	}

	// Print results
	fmt.Println("\nResults:")
	fmt.Println("========")
	printResults(result, *showCost, *showUsage)

	// Export to CSV if requested
	if *outputFile != "" {
		err := exportToCsv(result, *outputFile, *showCost, *showUsage)
		if err != nil {
			log.Fatalf("Error exporting to CSV: %v", err)
		}
		fmt.Printf("\nResults exported to %s\n", *outputFile)
	}
}

// ensureSsoLogin checks if an SSO session is active, and if not, initiates the login flow
func ensureSsoLogin(profileName string) error {
	// Build the AWS CLI command
	profile := ""
	if profileName != "" {
		profile = "--profile=" + profileName
	}

	// Check if SSO login is needed
	fmt.Println("Checking if SSO login is needed...")
	checkCmd := exec.Command("aws", "sts", "get-caller-identity", profile)
	if err := checkCmd.Run(); err != nil {
		fmt.Println("SSO session not found or expired. Launching browser for login...")
		
		// Run AWS SSO login command
		loginCmd := exec.Command("aws", "sso", "login", profile)
		loginCmd.Stdout = os.Stdout
		loginCmd.Stderr = os.Stderr
		
		if err := loginCmd.Run(); err != nil {
			return fmt.Errorf("failed to perform SSO login: %w", err)
		}
	} else {
		fmt.Println("Using existing SSO credentials")
	}
	
	return nil
}

// getInstanceUsage queries AWS Cost Explorer for instance usage/cost data
func getInstanceUsage(ce *costexplorer.Client, startMonth, endMonth string, instanceTypes []string, showCost, showUsage bool) ([]map[string]interface{}, error) {
	// Parse and validate date formats
	start, err := time.Parse(dateFormat, startMonth)
	if err != nil {
		return nil, fmt.Errorf("invalid start month format: %w", err)
	}

	end, err := time.Parse(dateFormat, endMonth)
	if err != nil {
		return nil, fmt.Errorf("invalid end month format: %w", err)
	}

	// Add one month to the end date to make it inclusive
	endDate := time.Date(end.Year(), end.Month()+1, 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, -1).Format("2006-01-02")
	
	startDate := start.Format("2006-01-02")

	// Expand instance type patterns
	expandedTypes := expandInstanceTypeFilter(instanceTypes)

	// Determine which metrics to request
	metrics := []string{}
	if showCost {
		metrics = append(metrics, "UnblendedCost")
	}
	if showUsage {
		metrics = append(metrics, "UsageQuantity")
	}
	if len(metrics) == 0 {
		metrics = append(metrics, "UsageQuantity") // Default to usage for checking existence
	}

	// Build Cost Explorer API request
	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: &startDate,
			End:   &endDate,
		},
		Granularity: types.GranularityMonthly,
		Metrics:     metrics,
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String("INSTANCE_TYPE"),
			},
		},
		Filter: &types.Expression{
			And: []types.Expression{
				{
					Dimensions: &types.DimensionValues{
						Key:    aws.String("SERVICE"),
						Values: []string{"Amazon Elastic Compute Cloud"},
					},
				},
				{
					Dimensions: &types.DimensionValues{
						Key:    aws.String("USAGE_TYPE"),
						Values: []string{"*BoxUsage*"},
					},
				},
				{
					Dimensions: &types.DimensionValues{
						Key:    aws.String("INSTANCE_TYPE"),
						Values: expandedTypes,
					},
				},
			},
		},
	}

	// Call the Cost Explorer API
	resp, err := ce.GetCostAndUsage(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost and usage data: %w", err)
	}

	// Process the response
	var results []map[string]interface{}
	for _, timePeriod := range resp.ResultsByTime {
		for _, group := range timePeriod.Groups {
			// Check if the instance type matches our patterns
			instanceType := *group.Keys[0]
			if matchesAnyPattern(instanceType, instanceTypes) {
				result := map[string]interface{}{
					"Period":       (*timePeriod.TimePeriod.Start)[:7], // YYYY-MM
					"InstanceType": instanceType,
				}

				if showCost {
					result["Cost"] = parseFloat(*group.Metrics["UnblendedCost"].Amount)
				}
				if showUsage {
					result["Usage"] = parseFloat(*group.Metrics["UsageQuantity"].Amount)
				} else if !showCost {
					// If neither cost nor usage requested, just show if there was usage
					result["HasUsage"] = parseFloat(*group.Metrics["UsageQuantity"].Amount) > 0
				}

				results = append(results, result)
			}
		}
	}

	return results, nil
}

// expandInstanceTypeFilter converts instance type patterns like "t3." to "*t3.*" for filtering
func expandInstanceTypeFilter(instanceTypes []string) []string {
	expandedTypes := make([]string, len(instanceTypes))
	for i, instType := range instanceTypes {
		if strings.HasSuffix(instType, ".") {
			expandedTypes[i] = "*" + instType + "*"
		} else {
			expandedTypes[i] = instType
		}
	}
	return expandedTypes
}

// matchesAnyPattern checks if an instance type matches any of the provided patterns
func matchesAnyPattern(instanceType string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchesPattern(instanceType, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern checks if an instance type matches a specific pattern
func matchesPattern(instanceType, pattern string) bool {
	if strings.HasSuffix(pattern, ".") {
		return strings.HasPrefix(instanceType, pattern[:len(pattern)-1])
	}
	return instanceType == pattern
}

// printResults formats and prints the instance usage/cost data
func printResults(results []map[string]interface{}, showCost, showUsage bool) {
	// TODO: Implement proper table formatting
	if len(results) == 0 {
		fmt.Println("No matching instances found")
		return
	}

	// Print header
	header := "Period      InstanceType "
	if showCost {
		header += "        Cost "
	}
	if showUsage {
		header += "        Usage "
	}
	if !showCost && !showUsage {
		header += "HasUsage "
	}
	fmt.Println(header)

	// Print data rows
	for _, result := range results {
		line := fmt.Sprintf("%-10s %-20s", result["Period"], result["InstanceType"])
		
		if showCost {
			line += fmt.Sprintf("$%-10.2f", result["Cost"])
		}
		if showUsage {
			line += fmt.Sprintf("%-10.2f", result["Usage"])
		}
		if !showCost && !showUsage {
			hasUsage := "No"
			if result["HasUsage"].(bool) {
				hasUsage = "Yes"
			}
			line += hasUsage
		}
		
		fmt.Println(line)
	}
}

// exportToCsv writes results to a CSV file
func exportToCsv(results []map[string]interface{}, filename string, showCost, showUsage bool) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write header
	header := "Period,InstanceType"
	if showCost {
		header += ",Cost"
	}
	if showUsage {
		header += ",Usage"
	}
	if !showCost && !showUsage {
		header += ",HasUsage"
	}
	fmt.Fprintln(f, header)

	// Write data rows
	for _, result := range results {
		line := fmt.Sprintf("%s,%s", result["Period"], result["InstanceType"])
		
		if showCost {
			line += fmt.Sprintf(",%.2f", result["Cost"])
		}
		if showUsage {
			line += fmt.Sprintf(",%.2f", result["Usage"])
		}
		if !showCost && !showUsage {
			hasUsage := "false"
			if result["HasUsage"].(bool) {
				hasUsage = "true"
			}
			line += "," + hasUsage
		}
		
		fmt.Fprintln(f, line)
	}

	return nil
}

// parseFloat safely converts a string to a float64
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// aws is a helper type for string pointers
type aws struct{}

func (a aws) String(v string) *string {
	return &v
}