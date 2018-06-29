package stats

import (
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
)

// GetEC2Usage gets EC2 usage for given session.
func GetEC2Usage(sess *session.Session) (ec2Usage map[string]string) {
	ec2Usage = make(map[string]string)

	svc := ec2.New(sess)
	// Get running instances
	respDescribeInstances, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
		},
	})
	if err != nil {
		ec2Usage["Running Instances"] = "0"
		fmt.Println(err.Error())
	} else {
		count := 0
		for i := 0; i < len(respDescribeInstances.Reservations); i++ {
			count += len(respDescribeInstances.Reservations[i].Instances)
		}
		ec2Usage["Running Instances"] = strconv.Itoa(count)
	}

	// Get volumes
	respDescribeVolumes, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		ec2Usage["EBS Volumes"] = "0"
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeVolumes.Volumes)
		ec2Usage["EBS Volumes"] = strconv.Itoa(count)
	}

	// Get AMIs
	respDescribeImages, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Owners: aws.StringSlice([]string{os.Getenv("AWS_ACCOUNT_ID")}),
	})
	if err != nil {
		ec2Usage["AMI Images"] = "0"
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeImages.Images)
		ec2Usage["AMI Images"] = strconv.Itoa(count)
	}

	// Get Snapshots
	respDescribeSnapshots, err := svc.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		OwnerIds: aws.StringSlice([]string{os.Getenv("AWS_ACCOUNT_ID")}),
	})
	if err != nil {
		ec2Usage["Snapshots"] = "0"
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeSnapshots.Snapshots)
		ec2Usage["Snapshots"] = strconv.Itoa(count)
	}

	// Get EIPs
	respDescribeAddresses, err := svc.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		ec2Usage["Elastic IPs"] = "0"
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeAddresses.Addresses)
		ec2Usage["Elastic IPs"] = strconv.Itoa(count)
	}

	elbSVC := elb.New(sess)
	respDescribeLoadBalancers, err := elbSVC.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		ec2Usage["Load Balancers"] = "0"
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeLoadBalancers.LoadBalancerDescriptions)
		ec2Usage["Load Balancers"] = strconv.Itoa(count)
	}

	return ec2Usage
}
