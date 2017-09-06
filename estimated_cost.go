package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func getEstimatedCost(startTime, endTime time.Time, estimatedCost chan<- float64) {
	svc := cloudwatch.New(sess)

	fmt.Println("Start time: ", startTime)
	fmt.Println("End time: ", endTime)

	params := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/Billing"),
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		MetricName: aws.String("EstimatedCharges"),
		Period:     aws.Int64(86400),
		Statistics: []*string{
			aws.String("Maximum"),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("Currency"),
				Value: aws.String("USD"),
			},
		},
	}

	resp, err := svc.GetMetricStatistics(params)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	jsonBody, _ := json.Marshal(resp)

	var result result
	json.Unmarshal(jsonBody, &result)
	sort.Sort(result.Datapoints)

	if len(result.Datapoints) > 0 {
		estimatedCost <- result.Datapoints[0].Maximum
	} else {
		estimatedCost <- 0
	}
}
