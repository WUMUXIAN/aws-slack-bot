package stats

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

// GetEstimatedCost calculates estimated cost for given session within specified period of time
func GetEstimatedCost(sess *session.Session, startTime, endTime time.Time) float64 {
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
		return 0
	}

	jsonBody, _ := json.Marshal(resp)

	var result result
	json.Unmarshal(jsonBody, &result)
	sort.Sort(result.Datapoints)

	if len(result.Datapoints) > 0 {
		return result.Datapoints[0].Maximum
	} else {
		return 0
	}
}
