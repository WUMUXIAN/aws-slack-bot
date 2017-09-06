package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/rds"
)

func getRDSUsage(sess *session.Session, startTime, endTime time.Time, RDSUsageChan chan map[string]string) {
	RDSUsage := make(map[string]string)

	svc := rds.New(sess)

	// List clusters
	respDescribeDBClusters, err := svc.DescribeDBClusters(&rds.DescribeDBClustersInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		RDSUsage["Clusters"] = strconv.Itoa(len(respDescribeDBClusters.DBClusters))
	}

	// List DB Instances
	respDescribeDBInstances, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		RDSUsage["Instances"] = strconv.Itoa(len(respDescribeDBInstances.DBInstances))
	}

	svcCloudWatch := cloudwatch.New(sess)

	// Get CPU Usage
	cpuUsage := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/RDS"), aws.String("CPUUtilization"), "Average", []*cloudwatch.Dimension{})[0]
	RDSUsage["CPU"] = fmt.Sprintf("%0.2f%%", cpuUsage)

	// Get Queries
	queries := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/RDS"), aws.String("Queries"), "Average", []*cloudwatch.Dimension{})[0]
	RDSUsage["Queries"] = fmt.Sprintf("%0.2f/Second", queries)

	// Get NetworkThroughput
	throughput := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/RDS"), aws.String("NetworkThroughput"), "Average", []*cloudwatch.Dimension{})[0]
	RDSUsage["NetworkThroughput"] = fmt.Sprintf("%s/Second", formatStorage(throughput))

	// Get Deadlocks
	deadlocks := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/RDS"), aws.String("Deadlocks"), "Average", []*cloudwatch.Dimension{})[0]
	RDSUsage["Deadlocks"] = fmt.Sprintf("%0.2f/Second", deadlocks)

	RDSUsageChan <- RDSUsage
}
