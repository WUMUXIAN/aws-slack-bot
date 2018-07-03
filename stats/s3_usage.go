package stats

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/s3"
)

// GetS3Usage gets the S3 usage for given session within specified period of time.
func GetS3Usage(sess *session.Session, startTime, endTime time.Time) (s3Usage map[string]string) {
	s3Usage = make(map[string]string)

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

	totalBytes := float64(0)
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
			totalBytes += sizeInBytes
		}
	}
	if totalBytes > 0 {
		s3Usage["_total size_"] = formatStorage(totalBytes)
	}
	return s3Usage
}
