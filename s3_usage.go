package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/s3"
)

func getS3Usage(sess *session.Session, startTime, endTime time.Time, s3UsageChan chan map[string]string) {
	s3Usage := make(map[string]string)

	svc := s3.New(sess)

	// List buckets
	buckets := make([]string, 0)
	respListBuckets, err := svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		for _, bucket := range respListBuckets.Buckets {
			buckets = append(buckets, aws.StringValue(bucket.Name))
		}
	}

	svcCloudWatch := cloudwatch.New(sess)

	demensionStandardStorage := &cloudwatch.Dimension{
		Name:  aws.String("StorageType"),
		Value: aws.String("StandardStorage"),
	}

	for _, bucket := range buckets {
		demensions := []*cloudwatch.Dimension{
			demensionStandardStorage,
			&cloudwatch.Dimension{
				Name:  aws.String("BucketName"),
				Value: aws.String(bucket),
			}}
		sizeInBytes := getMetricsStatistics(svcCloudWatch, startTime, endTime, aws.String("AWS/S3"), aws.String("BucketSizeBytes"), "Sum", demensions)[0]
		if sizeInBytes > 0 {
			s3Usage[bucket] = formatStorage(sizeInBytes)
		}
	}

	s3UsageChan <- s3Usage
}
