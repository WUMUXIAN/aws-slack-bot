// Package stats contains functions to gather usage stats of different services and consolidated billing estimations.
package stats

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

// GetCloudFrontUsage gets cloudfront usage for given session within specified period of time.
func GetCloudFrontUsage(sess *session.Session, startTime, endTime time.Time) (cloudFrontUsage map[string]string) {
	cloudFrontUsage = make(map[string]string)
	svc := cloudwatch.New(sess)

	interestedMetrics := make([]*cloudwatch.Metric, 0)
	respListMetrics, err := svc.ListMetrics(&cloudwatch.ListMetricsInput{
		Namespace: aws.String("AWS/CloudFront"),
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		for _, metrics := range respListMetrics.Metrics {
			if aws.StringValue(metrics.MetricName) == "Requests" || aws.StringValue(metrics.MetricName) == "BytesDownloaded" {
				interestedMetrics = append(interestedMetrics, metrics)
			}
		}
	}

	requests := float64(0)
	downloads := float64(0)
	for _, metrics := range interestedMetrics {
		if aws.StringValue(metrics.MetricName) == "Requests" {
			stats := getMetricsStatistics(svc, startTime, endTime, metrics.Namespace, metrics.MetricName, "Sum", metrics.Dimensions)
			reqs := float64(0)
			for i := 1; i < len(stats); i++ {
				reqs += stats[i]
			}
			if len(stats) > 1 {
				requests += reqs / float64(len(stats)-1)
			}
		} else {
			stats := getMetricsStatistics(svc, startTime, endTime, metrics.Namespace, metrics.MetricName, "Sum", metrics.Dimensions)
			dls := float64(0)
			for i := 1; i < len(stats); i++ {
				dls += stats[i]
			}
			if len(stats) > 1 {
				downloads += dls / float64(len(stats)-1)
			}
		}
	}
	if requests > 0 {
		cloudFrontUsage["Request Count"] = fmt.Sprintf("%0.2f/Day", requests)
	}
	if downloads > 0 {
		cloudFrontUsage["Downloaded Size"] = fmt.Sprintf("%s/Day", formatStorage(downloads))
	}

	return cloudFrontUsage
}
