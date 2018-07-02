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
		fmt.Println(err.Error())
	} else {
		count := 0
		for i := 0; i < len(respDescribeInstances.Reservations); i++ {
			count += len(respDescribeInstances.Reservations[i].Instances)
		}
		if count > 0 {
			ec2Usage["Running Instances"] = strconv.Itoa(count)
		}
	}

	// Get volumes
	respDescribeVolumes, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeVolumes.Volumes)
		if count > 0 {
			ec2Usage["EBS Volumes"] = strconv.Itoa(count)
		}
	}

	// Get AMIs
	respDescribeImages, err := svc.DescribeImages(&ec2.DescribeImagesInput{
		Owners: aws.StringSlice([]string{os.Getenv("AWS_ACCOUNT_ID")}),
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeImages.Images)
		if count > 0 {
			ec2Usage["AMI Images"] = strconv.Itoa(count)
		}
	}

	// Get Snapshots
	respDescribeSnapshots, err := svc.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		OwnerIds: aws.StringSlice([]string{os.Getenv("AWS_ACCOUNT_ID")}),
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeSnapshots.Snapshots)
		if count > 0 {
			ec2Usage["Snapshots"] = strconv.Itoa(count)
		}
	}

	// Get EIPs
	respDescribeAddresses, err := svc.DescribeAddresses(&ec2.DescribeAddressesInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeAddresses.Addresses)
		if count > 0 {
			ec2Usage["Elastic IPs"] = strconv.Itoa(count)
		}
	}

	elbSVC := elb.New(sess)
	respDescribeLoadBalancers, err := elbSVC.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		count := len(respDescribeLoadBalancers.LoadBalancerDescriptions)
		if count > 0 {
			ec2Usage["Load Balancers"] = strconv.Itoa(count)
		}
	}

	return ec2Usage
}
