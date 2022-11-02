package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"strings"
	"time"
)

type AWSInstance struct {
	Name       string
	Id         string
	Type       string
	Core       int
	HT         bool
	GP2Storage int
	GP3Storage int
}

type InstanceList map[string][]AWSInstance

const UNNAMEDINSTANCE = "unknown"
const GP2Price = 0.088    // default provisioning
const GP3Price = 0.11     // default provisioning
const TrafficPrice = 0.07 // estimated from previous bills

// GetEBSCostForMonth returns the price of the gp2 and gp3 storage with it's default provisioned pricing.
func (s AWSInstance) GetEBSCostForMonth() float64 {
	return float64(s.GP2Storage)*GP2Price + float64(s.GP3Storage)*GP3Price
}

// GetInstanceCostForHour is a very narrow-minded function that returns instance type costs.
// Only works for Canada instances.
// Only works for on-demand instances.
// Does not query AWS.
// Only works for pre-defined instances.
func (s AWSInstance) GetInstanceCostForHour() float64 {
	instanceCostsInCanada := map[string]float64{
		"c5a.large":    0.084,
		"db.gp2":       0.253,
		"db.t4g.small": 0.07,
		"m5a.2xlarge":  0.384,
		"m5a.large":    0.096,
		"m5a.xlarge":   0.192,
		"m6i.4xlarge":  0.856,
		"m6i.xlarge":   0.214,
		"r5a.4xlarge":  0.992,
		"r5a.large":    0.124,
		"r5a.xlarge":   0.248,
		"t3.2xlarge":   0.3712,
		"t3a.large":    0.0835,
		"t3a.medium":   0.0418,
		"t3a.nano":     0.0052,
		"t3a.small":    0.0209,
		"t3a.xlarge":   0.167,
		"i4i.2xlarge":  0.757,
		"i4i.xlarge":   0.378,
		"i4i.large":    0.189,
		"m6i.large":    0.107,
		"t2.micro":     0.0128,
	}
	if result, ok := instanceCostsInCanada[s.Type]; ok {
		return result
	}
	panic(fmt.Sprintf("did not find cost for instance type %s", s.Type))
	return -10000000000.0
}

// GetInstanceCostFor30Days is a convenience function that calls GetInstanceCostForHour and calculates cost for 30 days.
func (s AWSInstance) GetInstanceCostFor30Days() float64 {
	return s.GetInstanceCostForHour() * 24 * 30
}

// GetTrafficInGB returns the total traffic in the past 30 days for the node in gigabyte.
// As described in the Amazon documentation for GetMetricStatistics:
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/viewing_metrics_with_cloudwatch.html
// The number of bytes sent out by the instance on all network interfaces.
// This metric identifies the volume of outgoing network traffic from a single instance.
//
// The number reported is the number of bytes sent during the period.
// If you are using basic (5-minute) monitoring and the statistic is Sum,
// you can divide this number by 300 to find Bytes/second.
// If you have detailed (1-minute) monitoring and the statistic is Sum, divide it by 60.
//
// Units: Bytes
func (s AWSInstance) GetTrafficInGB(cfg aws.Config) float64 {
	cwsvc := cloudwatch.NewFromConfig(cfg)
	input := &cloudwatch.GetMetricStatisticsInput{
		StartTime:  aws.Time(time.Now().Add(-1 * 30 * 24 * time.Hour)), // 30 days ago
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String("NetworkOut"),
		Namespace:  aws.String("AWS/EC2"),
		Period:     aws.Int32(5 * 24 * 30), // Sampling every 5 minutes, over the period of 30 days
		Dimensions: []cwtypes.Dimension{{Name: aws.String("InstanceId"), Value: aws.String(s.Id)}},
		Statistics: []cwtypes.Statistic{cwtypes.StatisticSum},
	}
	output, err := cwsvc.GetMetricStatistics(context.TODO(), input)
	if err != nil {
		panic("Got an error retrieving information about your Amazon CloudWatch metrics, " + err.Error())
	}
	total := 0.0
	for _, d := range output.Datapoints {
		total += *d.Sum
	}
	total /= 1024 * 1024 * 1024 // gigabyte

	return total
}

// GetTrafficCostFor30Days is a convenience function that gets the amount of traffic spent in the last 30 days
// and calculates its price.
func (s AWSInstance) GetTrafficCostFor30Days(cfg aws.Config) (trafficcost float64, traffic float64) {
	traffic = s.GetTrafficInGB(cfg)
	trafficcost = traffic * TrafficPrice
	return
}

// NewInstancesByGroup organizes the available instances by their network into a HashMap.
func NewInstancesByGroup(cfg aws.Config) (instances InstanceList) {
	// Get all EC2 node descriptions
	ec2nodes := GetEC2Instances(cfg)

	instances = InstanceList{UNNAMEDINSTANCE: nil}
	for _, n := range ec2nodes.Reservations {
		for counter, i := range n.Instances {
			// Find AWS name
			name := GetNameFromTags(i.Tags, fmt.Sprintf("%s%d", UNNAMEDINSTANCE, counter))

			// Find all block devices
			gp2Storage, gp3Storage := GetBlockdeviceSizes(cfg, i.BlockDeviceMappings)

			group := strings.Trim(name, "0123456789")
			node := AWSInstance{
				Name:       name,
				Id:         *i.InstanceId,
				Type:       string(i.InstanceType),
				Core:       int(*i.CpuOptions.CoreCount),
				HT:         *i.CpuOptions.ThreadsPerCore > 0,
				GP2Storage: gp2Storage,
				GP3Storage: gp3Storage,
			}
			if _, ok := instances[group]; ok {
				instances[group] = append(instances[group], node)
			} else {
				instances[group] = []AWSInstance{node}
			}
		}
	}
	if instances[UNNAMEDINSTANCE] == nil {
		delete(instances, UNNAMEDINSTANCE)
	} else {
		if len(instances[UNNAMEDINSTANCE]) == 0 {
			delete(instances, UNNAMEDINSTANCE)
		} else {
			fmt.Printf("WARNING: %d instances found with no name tag(s): %v\n", len(instances[UNNAMEDINSTANCE]), instances[UNNAMEDINSTANCE])
		}
	}
	return
}
