package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func getPossibleAlarms(client *cloudwatch.Client, instance awsInstance) ([][]cwTypes.Dimension, error) {
	var metrics [][]cwTypes.Dimension
	filters := cloudwatch.ListMetricsInput{}
	filters.Dimensions = []cwTypes.DimensionFilter{
		{
			Name:  aws.String("InstanceId"),
			Value: aws.String(instance.ID),
		},
	}
	switch {
	case instance.OS == "Windows":
		filters.MetricName = aws.String("LogicalDisk % Free Space")
	case instance.OS == "Linux/UNIX":
		filters.MetricName = aws.String("disk_used_percent")
	default:
		return metrics, fmt.Errorf("Unknown type of OS: %s for %s", instance.OS, instance.ID)
	}
	r, err := client.ListMetrics(context.TODO(), &filters)
	if err != nil {
		return metrics, err
	}
	if len(r.Metrics) == 0 {
		return metrics, fmt.Errorf("No metrics defined for %s", instance.Name)
	}
	for _, i := range r.Metrics {
		if instance.OS != "Windows" {
			if checkFSType(i.Dimensions) {
				metrics = append(metrics, i.Dimensions)
			}
		} else if checkDiskInstance(i.Dimensions) {
			metrics = append(metrics, i.Dimensions)
		}
	}
	if len(metrics) != 0 {
		return metrics, nil
	}
	return metrics, fmt.Errorf("No metrics defined for %s", instance.Name)
}

// Returns a slice of the running instances
func getRunningInstanceIDs(client *ec2.Client) ([]awsInstance, error) {
	var instances []awsInstance
	t, e := client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		Filters: []ec2Types.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []string{
					"running",
				},
			},
		},
	})
	if e != nil {
		return instances, e
	}
	for _, i := range t.Reservations {
		for _, v := range i.Instances {
			instances = append(instances, awsInstance{ID: *v.InstanceId, OS: *v.PlatformDetails, Name: getNameTag(v.Tags, *v.InstanceId)})
		}
	}
	return instances, nil
}

// Get windows disk instance. If it exists, it returns true, otherwise it returns false.
// This is used to filter out metrics that appear in the API, but we don't care about
func checkDiskInstance(labels []cwTypes.Dimension) bool {
	for _, l := range labels {
		if *l.Name == "instance" {
			return true
		}
	}
	return false
}

// Return a bool if the the fstype is xfs or ext
func checkFSType(labels []cwTypes.Dimension) bool {
	for _, l := range labels {
		if *l.Name == "fstype" {
			if strings.Contains(*l.Value, "ext") || *l.Value == "xfs" {
				return true
			}
		}
	}
	return false
}
func getNameTag(labels []ec2Types.Tag, instID string) string {
	for _, l := range labels {
		if *l.Key == "Name" {
			return *l.Value
		}
	}
	return instID
}
