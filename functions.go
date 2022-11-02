package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"os"
)

// GetAWSConfig returns an AWS configuration struct initialized for a specified region.
func GetAWSConfig(region string) aws.Config {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	return cfg
}

var _ec2instances *ec2.DescribeInstancesOutput

// GetEC2Instances queries AWS for the currently available EC2 instances.
func GetEC2Instances(cfg aws.Config) ec2.DescribeInstancesOutput {
	if _ec2instances == nil {
		ec2svc := ec2.NewFromConfig(cfg)
		input := &ec2.DescribeInstancesInput{}
		fmt.Fprintf(os.Stderr, "Getting EC2 instance descriptions...\n")
		var err error
		_ec2instances, err = ec2svc.DescribeInstances(context.TODO(), input)
		if err != nil {
			panic("Got an error retrieving information about your Amazon EC2 instances, " + err.Error())
		}
		if _ec2instances == nil {
			panic("Empty result querying EC2 instances.")
		}
		if _ec2instances.NextToken != nil {
			panic("You have too many instances and paging is not implemented.")
		}
	}
	return *_ec2instances
}

var _volumes *ec2.DescribeVolumesOutput

// GetVolumes queries AWS for all the EBS volumes.
func GetVolumes(cfg aws.Config) ec2.DescribeVolumesOutput {
	if _volumes == nil {
		ec2svc := ec2.NewFromConfig(cfg)
		input := &ec2.DescribeVolumesInput{}
		fmt.Fprintf(os.Stderr, "Getting volume descriptions...\n")
		var err error
		_volumes, err = ec2svc.DescribeVolumes(context.TODO(), input)
		if err != nil {
			panic("Got an error retrieving information about your Amazon EC2 volumes, " + err.Error())
		}
		if _volumes == nil {
			panic("Empty result querying EC2 volumes.")
		}
		if _volumes.NextToken != nil {
			panic("You have too many volumes and paging is not implemented.")
		}
	}
	return *_volumes
}

// GetVolumeById returns a volume description from the list of all volumes.
func GetVolumeById(volumes ec2.DescribeVolumesOutput, volumeId string) ec2types.Volume {
	for _, v := range volumes.Volumes {
		if *v.VolumeId == volumeId {
			return v
		}
	}
	panic("volume " + volumeId + " not found")
}

// GetNameFromTags gets the Name tag from a list of tags or returns a default Name if there is no such tag.
func GetNameFromTags(tags []ec2types.Tag, defaultName string) string {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			if *tag.Value != "" {
				return *tag.Value
			}
		}
	}
	return defaultName
}

// GetBlockdeviceSizes queries AWS for all gp2 and gp3 volume sizes and
func GetBlockdeviceSizes(cfg aws.Config, bds []ec2types.InstanceBlockDeviceMapping) (gp2Storage int, gp3Storage int) {
	// Get all volumes
	volumes := GetVolumes(cfg)
	// Go through devices and get their size.
	for _, b := range bds {
		volume := GetVolumeById(volumes, *b.Ebs.VolumeId)
		switch volume.VolumeType {
		case ec2types.VolumeTypeGp2:
			gp2Storage += int(*volume.Size)
		case ec2types.VolumeTypeGp3:
			gp3Storage += int(*volume.Size)
		default:
			panic("volume type " + volume.VolumeType + "not support. Tell Greg to implement it...")
		}
	}
	return
}
