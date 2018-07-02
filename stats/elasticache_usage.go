package stats

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elasticache"
)

// GetElasticacheUsage gets elasticache usage for given sessions within specified period of time.
func GetElasticacheUsage(sess *session.Session, startTime, endTime time.Time) (elasticacheUsage map[string]string) {
	elasticacheUsage = make(map[string]string)

	svc := elasticache.New(sess)
	respDescribeReplicationGroups, err := svc.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeReplicationGroups.ReplicationGroups)
		if count > 0 {
			elasticacheUsage["Replication Groups"] = strconv.Itoa(count)
		}

	}

	// List clusters
	respDescribeCacheClusters, err := svc.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeCacheClusters.CacheClusters)
		if count > 0 {
			elasticacheUsage["Clusters"] = strconv.Itoa(count)
		}
		if len(respDescribeCacheClusters.CacheClusters) > 0 {
			nodes := 0
			for _, elasticacheCluster := range respDescribeCacheClusters.CacheClusters {
				nodes += int(aws.Int64Value(elasticacheCluster.NumCacheNodes))
			}
			if nodes > 0 {
				elasticacheUsage["Nodes"] = strconv.Itoa(nodes)
			}
		}

	}

	svcCloudWatch := cloudwatch.New(sess)

	// Get CPU Usage
	cpuUsage := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/ElastiCache"), aws.String("CPUUtilization"), "Average", []*cloudwatch.Dimension{})[0]
	if cpuUsage > 0 {
		elasticacheUsage["CPU"] = fmt.Sprintf("%0.2f%%", cpuUsage)
	}

	// Get Queries
	bytes := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/ElastiCache"), aws.String("BytesUsedForCache"), "Average", []*cloudwatch.Dimension{})[0]
	if bytes > 0 {
		elasticacheUsage["Cache Size"] = formatStorage(bytes)
	}

	return elasticacheUsage
}
