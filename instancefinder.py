import boto3
import argparse
from datetime import datetime
import calendar
import pandas as pd
import re

def expand_instance_type_filter(instance_types):
    """
    Convert instance type patterns into appropriate filter values
    e.g., 't3.' becomes '*t3.*'
    """
    expanded_types = []
    for inst_type in instance_types:
        if inst_type.endswith('.'):
            expanded_types.append(f'*{inst_type}*')
        else:
            expanded_types.append(inst_type)
    return expanded_types

def get_instance_usage(start_month, end_month, instance_types, show_cost=False, show_usage=False):
    """
    Retrieve EC2 instance usage/cost data for specified parameters
    
    Args:
        start_month (str): Start month in YYYY-MM format
        end_month (str): End month in YYYY-MM format
        instance_types (list): List of instance types or patterns to analyze
        show_cost (bool): Whether to show cost data
        show_usage (bool): Whether to show usage data
    """
    
    # Convert dates to first/last of month
    start_date = f"{start_month}-01"
    _, last_day = calendar.monthrange(int(end_month[:4]), int(end_month[5:7]))
    end_date = f"{end_month}-{last_day}"

    client = boto3.client('ce')

    # Expand instance type patterns and create filter
    expanded_types = expand_instance_type_filter(instance_types)
    
    filters = {
        'And': [
            {'Dimensions': {'Key': 'SERVICE', 'Values': ['Amazon Elastic Compute Cloud']}},
            {'Dimensions': {'Key': 'USAGE_TYPE', 'Values': ['*BoxUsage*']}},
            {'Dimensions': {'Key': 'INSTANCE_TYPE', 'Values': expanded_types}}
        ]
    }

    # Determine which metrics to request
    metrics = []
    if show_cost:
        metrics.append('UnblendedCost')
    if show_usage:
        metrics.append('UsageQuantity')
    if not metrics:
        metrics = ['UsageQuantity']  # Default to usage for checking existence

    try:
        response = client.get_cost_and_usage(
            TimePeriod={
                'Start': start_date,
                'End': end_date
            },
            Granularity='MONTHLY',
            Metrics=metrics,
            GroupBy=[
                {'Type': 'DIMENSION', 'Key': 'INSTANCE_TYPE'},
            ],
            Filter=filters
        )

        # Process the response
        results = []
        for time_period in response['ResultsByTime']:
            for group in time_period['Groups']:
                # Check if the instance type matches our patterns
                instance_type = group['Keys'][0]
                if any(matches_pattern(instance_type, pattern) for pattern in instance_types):
                    result = {
                        'Period': time_period['TimePeriod']['Start'][:7],  # YYYY-MM
                        'InstanceType': instance_type
                    }
                    
                    if show_cost:
                        result['Cost'] = float(group['Metrics']['UnblendedCost']['Amount'])
                    if show_usage:
                        result['Usage'] = float(group['Metrics']['UsageQuantity']['Amount'])
                    elif not show_cost:
                        # If neither cost nor usage requested, just show if there was usage
                        result['HasUsage'] = float(group['Metrics']['UsageQuantity']['Amount']) > 0

                    results.append(result)

        return pd.DataFrame(results)

    except Exception as e:
        print(f"Error occurred: {str(e)}")
        return None

def matches_pattern(instance_type, pattern):
    """
    Check if instance type matches the pattern
    e.g., 't3.micro' matches 't3.'
    """
    if pattern.endswith('.'):
        return instance_type.startswith(pattern[:-1])
    return instance_type == pattern

def main():
    parser = argparse.ArgumentParser(description='Analyze AWS EC2 instance usage')
    parser.add_argument('--start', required=True, help='Start month (YYYY-MM)')
    parser.add_argument('--end', required=True, help='End month (YYYY-MM)')
    parser.add_argument('--instances', required=True, nargs='+', 
                        help='List of instance types (use type. for all of that type, e.g., t3.)')
    parser.add_argument('--show-cost', action='store_true', help='Show cost data')
    parser.add_argument('--show-usage', action='store_true', help='Show usage data')
    parser.add_argument('--output', help='Output file path (CSV)')

    args = parser.parse_args()

    # Validate date format
    try:
        datetime.strptime(args.start, '%Y-%m')
        datetime.strptime(args.end, '%Y-%m')
    except ValueError:
        print("Error: Dates must be in YYYY-MM format")
        return

    df = get_instance_usage(
        args.start,
        args.end,
        args.instances,
        args.show_cost,
        args.show_usage
    )

    if df is not None:
        # Sort by Period and InstanceType
        df = df.sort_values(['Period', 'InstanceType'])
        
        # Print results
        print("\nResults:")
        print("========")
        print(df.to_string(index=False))

        # Export if output file specified
        if args.output:
            df.to_csv(args.output, index=False)
            print(f"\nResults exported to {args.output}")

if __name__ == "__main__":
    main()
