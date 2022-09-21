package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"gopkg.in/alecthomas/kingpin.v2"
)

var logger = &logrus.Logger{
	Out:   os.Stdout,
	Level: logrus.InfoLevel,
	Formatter: &easy.Formatter{
		LogFormat: "%msg%\n",
	},
}
var (
	debug = kingpin.Flag("debug", "If enabled, aws-check-alarm will go into debug mode. This will display all instances without cloudwatch, and all metrics that already have alarms set.").Short('d').Bool()
	yaos  = kingpin.Flag("yes", "Automatically generate all possible alarms.").Short('y').Bool()
	noo   = kingpin.Flag("no", "Skip prompt for generating alarms.").Short('n').Bool()
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
			logger.Error(e)
			continue
		}

		if len(res.MetricAlarms) == 0 {
			logger.Warnf("No alarms exist for %s %s on %s", *t.MetricName, getLocationName(t.Dimensions), i.Name)
			if handleInput() {
				if err := createAlarm(&client, *i, t); err != nil {
					logger.Error(err)
				}
			}
		} else {
			logger.Debugf("Metric exists for %s %s on %s", *t.MetricName, getLocationName(t.Dimensions), i.Name)
		}
	}
}

func handleInput() bool {
	if *noo {
		return false
	}
	if *yaos {
		logger.Info("Adding alarm")
		return true
	}
	logger.Info("Would you like to auto-generate a new alarm? (y/n)")
	var resp string
	fmt.Scanln(&resp)
	switch {
	case strings.ToLower(string(resp[0])) == "y":
		logger.Info("Adding alarm")
		return true
	case strings.ToLower(string(resp[0])) == "n":
		logger.Info("Will not generate alarm")
	default:
		logger.Error("Invalid option")
		handleInput()
	}
	return false

}

func main() {
	kingpin.Parse()
	if *debug {
		logger.SetLevel(logrus.DebugLevel)
	}
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		logger.Fatal(err)
	}
	// Create an Amazon ec2 service client
	ec2Client := ec2.NewFromConfig(cfg)
	cwClient := cloudwatch.NewFromConfig(cfg)

	runningInstances, err := getRunningInstanceIDs(ec2Client)
	if err != nil {
		logger.Fatal(err)
	}
	for _, i := range runningInstances {
		i, err := getPossibleAlarms(cwClient, i)
		if err != nil {
			logger.Debugln(err, "Skipping...")
			continue
		}
		i.CheckMetrics(*cwClient)
	}
}
