# AWS Instance Finder

<!-- SPDX-License-Identifier: MIT -->
<!-- SPDX-FileCopyrightText: Copyright 2025, Scott Friedman and Project Contributors -->

A Go utility for analyzing AWS EC2 instance usage and cost across your AWS account.

## Overview

AWS Instance Finder helps you identify which EC2 instance types are running in your AWS account, analyze usage patterns, and track costs over time. This tool is particularly useful for:

- Cost optimization - identifying unused or underutilized instances
- Compliance auditing - ensuring only approved instance types are in use
- Migration planning - finding instances of specific families to target for upgrades
- Budget tracking - analyzing instance costs across time periods

## Features

- Find which EC2 instance types are being used in your AWS account
- Analyze usage patterns over time with monthly granularity
- Filter instances by type or family (e.g., t3.micro or all t3.* instances)
- Show cost data for specific instance types
- Export results to CSV for further analysis
- Multiple authentication methods, including browser-based AWS SSO

## Installation

### Prerequisites

- **AWS CLI** is required if you plan to use the SSO authentication option (`--sso` flag)
- No special prerequisites for other authentication methods

### From Binary

Download the appropriate binary for your platform from the [releases page](https://github.com/scttfrdmn/aws-instance-finder/releases):

- **Windows**: `instancefinder-windows-amd64-v0.9.0.zip` or `instancefinder-windows-arm64-v0.9.0.zip`
- **macOS**: `instancefinder-darwin-amd64-v0.9.0.tar.gz` or `instancefinder-darwin-arm64-v0.9.0.tar.gz`
- **Linux**: `instancefinder-linux-amd64-v0.9.0.tar.gz` or `instancefinder-linux-arm64-v0.9.0.tar.gz`

Make the binary executable (Linux/macOS):

```bash
chmod +x instancefinder-*
```

### From Source

```bash
# Clone the repository
git clone https://github.com/scttfrdmn/aws-instance-finder.git
cd aws-instance-finder

# Build the binary
go build -o instancefinder ./instancefinder.go

# Make it executable (Linux/Mac)
chmod +x instancefinder
```

## Usage

```bash
./instancefinder --start YYYY-MM --end YYYY-MM --instances TYPE1,TYPE2 [OPTIONS]
```

### Arguments

- `--start`: Start month in YYYY-MM format
- `--end`: End month in YYYY-MM format
- `--instances`: Comma-separated list of instance types to analyze
  - Use `t3.` to match all t3 instance types
  - Use specific types like `t3.micro` for exact matches
- `--show-cost`: Include cost data in results
- `--show-usage`: Include usage data in results
- `--output`: Path to save results as CSV
- `--sso`: Use AWS SSO authentication (requires AWS CLI)
- `--profile`: AWS profile name to use
- `--version`: Show version information

### Authentication Methods

This tool supports multiple authentication methods:

1. **AWS SSO (Recommended for most users)**:
   - Use `--sso` flag to enable SSO authentication
   - The tool will automatically launch your browser to authenticate if needed
   - For first-time use, you'll be prompted to log in through your browser
   - Subsequent runs will reuse your SSO session until it expires
   - **Requires AWS CLI to be installed** as it uses the CLI for the browser-based authentication flow

2. **AWS Profile**:
   - Use `--profile=<name>` to use a specific profile from your AWS config
   - Works with both credentials files and SSO profiles

3. **Default Credential Provider Chain**:
   - If no authentication options are specified, the AWS SDK's default credential provider chain is used (environment variables, shared credentials files, EC2 instance profiles, etc.)

### Examples

Find if any t3 instances were used in Q1 2023 using SSO authentication:
```bash
./instancefinder --start 2023-01 --end 2023-03 --instances t3. --sso
```

Analyze costs for specific instance types using a named profile:
```bash
./instancefinder --start 2023-01 --end 2023-12 --instances t3.micro,m5.large --show-cost --profile production
```

Export usage data to CSV:
```bash
./instancefinder --start 2023-01 --end 2023-12 --instances t3.,m5. --show-usage --output instance_usage.csv
```

## AWS Permissions Required

> ⚠️ **IMPORTANT:** This tool requires Cost Explorer API access, which is not included in most default IAM policies.

Your AWS credentials need specific permissions to access the Cost Explorer API. Without these permissions, the tool will fail with an "AccessDeniedException" error.

### Required IAM Policy

Create a custom IAM policy with the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ce:GetCostAndUsage"
            ],
            "Resource": "*"
        }
    ]
}
```

### How to Apply the Policy

1. **For IAM users or roles:**
   - In the AWS Management Console, go to IAM
   - Create a new policy with the JSON above
   - Attach this policy to any IAM users/roles that will use this tool

2. **For SSO users:**
   - In AWS IAM Identity Center, create a Permission Set that includes the policy above
   - Assign this Permission Set to users or groups who need to use this tool

3. **For temporary credentials:**
   - Ensure that the IAM role you assume has the Cost Explorer permissions attached

### Cost Explorer API Activation

Note that Cost Explorer must be activated in your AWS account before the API can be used. If you haven't used Cost Explorer before, visit the AWS Cost Management console first to enable it.

## Building from Source for Multiple Platforms

See the included `build.sh` script for building binaries for multiple platforms.

```bash
./build.sh
```

## Python Version

A Python implementation is also available in the `python/` directory.

## License

This project is licensed under the MIT License - see the LICENSE file for details.