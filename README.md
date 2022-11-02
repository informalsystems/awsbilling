# AWS Billing
## Description
This is a crude little tool to get a monthly picture of how much we paid for our nodes in AWS.

The prices are hard-coded to the Canada region to save on development time  and instead of pin-point accuracy, the code goes back 30 days to find out what nodes we have with what traffic.

The biggest benefit is that it estimates the traffic usage and its price per-node which is hard to correlate to networks in the AWS Cost Explorer.

The output is a CSV-compatible comma-separated list of entries grouped by the network they belong to.
It also indicates some costs that are present in AWS but the code doesn't automatically pull it for now.

We can work on it for the future but right now this is a good enough estimate for once-a-month to see our costs.

## Execution
Build the binary:
```
go build
```

Run:
```
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
./awsbilling
```

If you use `aws-vault` you can run:
```
aws-vault exec <credential-name> -- ./awsbilling
```

You can directly send the output to a CSV file.
