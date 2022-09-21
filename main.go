package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/sirupsen/logrus"
)

type awsInstance struct {
	ID           string
	Name         string
	OS           string
	MetricLabels [][]cwTypes.Dimension
}

func (i *awsInstance) CheckMetrics(client cloudwatch.Client) string {
	var displaySTR string
	for _, t := range i.MetricLabels {
		for _, d := range t {

			displaySTR += *d.Name + ": " + *d.Value + "\t"
		}
	}
	return displaySTR
}

func main() {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		logrus.Fatal(err)
	}
	// Create an Amazon ec2 service client
	ec2Client := ec2.NewFromConfig(cfg)
	cwClient := cloudwatch.NewFromConfig(cfg)

	runningInstances, err := getRunningInstanceIDs(ec2Client)
	if err != nil {
		logrus.Fatal(err)
	}
	for _, i := range runningInstances {
		dimensions, err := getPossibleAlarms(cwClient, i)
		if err != nil {
			logrus.Warnln(err, "Skipping...")
			continue
		}
		i.MetricLabels = dimensions
		logrus.Info(i.CheckMetrics(*cwClient))
	}
}
