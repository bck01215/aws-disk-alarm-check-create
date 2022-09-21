package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/sirupsen/logrus"
)

type awsInstance struct {
	ID      string
	Name    string
	OS      string
	Metrics []cwTypes.Metric
}

func (i *awsInstance) CheckMetrics(client cloudwatch.Client) {

	for _, t := range i.Metrics {
		res, e := client.DescribeAlarmsForMetric(context.TODO(), &cloudwatch.DescribeAlarmsForMetricInput{
			MetricName: t.MetricName,
			Namespace:  t.Namespace,
			Dimensions: t.Dimensions,
		})
		if e != nil {
			logrus.Error(e)
			continue
		}

		if len(res.MetricAlarms) == 0 {
			logrus.Warnf("No alarms exist for %s %s on %s", *t.MetricName, getLocationName(t.Dimensions), i.Name)
			if handleInput() {
				if err := createAlarm(&client, *i, t); err != nil {
					logrus.Error(err)
				}
			}
		} else {
			logrus.Debugf("Metric exists for %s %s on %s", *t.MetricName, getLocationName(t.Dimensions), i.Name)
		}
	}
}

func handleInput() bool {
	logrus.Info("Would you like to auto-generate a new alarm? (y/n)")
	var resp string
	fmt.Scanln(&resp)
	switch {
	case strings.ToLower(string(resp[0])) == "y":
		logrus.Info("Adding alarm")
		return true
	case strings.ToLower(string(resp[0])) == "n":
		logrus.Info("Will not generate alarm")
	default:
		logrus.Error("Invalid option")
		handleInput()
	}
	return false

}

func main() {
	// Load the Shared AWS Configuration (~/.aws/config)
	logrus.SetLevel(logrus.DebugLevel)
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
		i, err := getPossibleAlarms(cwClient, i)
		if err != nil {
			logrus.Warnln(err, "Skipping...")
			continue
		}
		i.CheckMetrics(*cwClient)
	}
}
