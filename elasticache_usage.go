package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elasticache"
)

func getElasticacheUsage(sess *session.Session, startTime, endTime time.Time, elasticacheUsageChan chan map[string]string) {
	elasticacheUsage := make(map[string]string)

	svc := elasticache.New(sess)
	respDescribeReplicationGroups, err := svc.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		elasticacheUsage["Replication Groups"] = strconv.Itoa(len(respDescribeReplicationGroups.ReplicationGroups))
	}

	// List clusters
	respDescribeCacheClusters, err := svc.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		elasticacheUsage["Clusters"] = strconv.Itoa(len(respDescribeCacheClusters.CacheClusters))
		if len(respDescribeCacheClusters.CacheClusters) > 0 {
			nodes := 0
			for _, elasticacheCluster := range respDescribeCacheClusters.CacheClusters {
				nodes += int(aws.Int64Value(elasticacheCluster.NumCacheNodes))
			}
			elasticacheUsage["Nodes"] = strconv.Itoa(nodes)
		}

	}

	svcCloudWatch := cloudwatch.New(sess)

	// Get CPU Usage
	cpuUsage := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/ElastiCache"), aws.String("CPUUtilization"), "Average", []*cloudwatch.Dimension{})[0]
	elasticacheUsage["CPU"] = fmt.Sprintf("%0.2f%%", cpuUsage)

	// Get Queries
	bytes := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/ElastiCache"), aws.String("BytesUsedForCache"), "Average", []*cloudwatch.Dimension{})[0]
	elasticacheUsage["Cache Size"] = formatStorage(bytes)

	elasticacheUsageChan <- elasticacheUsage
}
