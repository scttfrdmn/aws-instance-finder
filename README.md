# AWS Instance Finder

<!-- SPDX-License-Identifier: MIT -->
<!-- SPDX-FileCopyrightText: Copyright 2025, Scott Friedman and Project Contributors -->

A Python utility for analyzing AWS EC2 instance usage and cost across your AWS account.

## Features

- Find which EC2 instance types are being used in your AWS account
- Analyze usage patterns over time with monthly granularity
- Filter instances by type or family (e.g., t3.micro or all t3.* instances)
- Show cost data for specific instance types
- Export results to CSV for further analysis

## Requirements

- Python 3.6+
- boto3
- pandas
- AWS credentials configured with access to Cost Explorer API

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/aws-instance-finder.git
cd aws-instance-finder

# Install dependencies
pip install boto3 pandas
```

## Usage

```bash
python instancefinder.py --start YYYY-MM --end YYYY-MM --instances [INSTANCE_TYPES] [OPTIONS]
```

### Arguments

- `--start`: Start month in YYYY-MM format
- `--end`: End month in YYYY-MM format
- `--instances`: One or more instance types to analyze
  - Use `t3.` to match all t3 instance types
  - Use specific types like `t3.micro` for exact matches
- `--show-cost`: Include cost data in results
- `--show-usage`: Include usage data in results
- `--output`: Path to save results as CSV
- `--profile`: AWS profile name to use (optional)

### Examples

Find if any t3 instances were used in Q1 2023:
```bash
python instancefinder.py --start 2023-01 --end 2023-03 --instances t3.
```

Analyze costs for specific instance types:
```bash
python instancefinder.py --start 2023-01 --end 2023-12 --instances t3.micro m5.large --show-cost
```

Export usage data to CSV:
```bash
python instancefinder.py --start 2023-01 --end 2023-12 --instances t3. m5. --show-usage --output instance_usage.csv
```

Using a specific AWS profile:
```bash
python instancefinder.py --start 2023-01 --end 2023-12 --instances t3. --profile production
```

## Permissions Required

Your AWS credentials need access to the Cost Explorer API. The minimum IAM permissions required are:
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

## License

This project is licensed under the MIT License - see the LICENSE file for details.