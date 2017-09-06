package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/robfig/cron"
)

var slackWebhookURL string

// Channels
var (
	_ec2UsageChan         chan map[string]string
	_s3UsageChan          chan map[string]string
	_cloudFrontUsageChan  chan map[string]string
	_rdsUsageChan         chan map[string]string
	_elasticacheUsageChan chan map[string]string
	_estimatedCostLast    chan float64
	_estimatedCostCurrent chan float64

	_msgSent chan bool
)

type result struct {
	Datapoints datapoints
}

var sess *session.Session

func init() {
	sess = session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
}

func getSortedKeySlice(m map[string]string) []string {
	keys := make([]string, 0)
	for key := range m {
		keys = append(keys, key)
	}
	sort.Sort(sort.StringSlice(keys))
	return keys
}

func messageSlack() {
	<-_msgSent

	go getEC2Usage(sess, _ec2UsageChan)

	now := time.Now().UTC()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()
	firstDayOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

	go getS3Usage(sess, firstDayOfMonth, lastDayOfMonth, _s3UsageChan)
	go getCloudFrontUsage(sess, firstDayOfMonth, lastDayOfMonth, _cloudFrontUsageChan)
	go getRDSUsage(sess, firstDayOfMonth, lastDayOfMonth, _rdsUsageChan)
	go getElasticacheUsage(sess, firstDayOfMonth, lastDayOfMonth, _elasticacheUsageChan)
	go getEstimatedCost(firstDayOfMonth, lastDayOfMonth, _estimatedCostCurrent)

	firstDayOfMonth = firstDayOfMonth.AddDate(0, -1, 0)
	lastDayOfMonth = firstDayOfMonth.AddDate(0, 1, -1)
	go getEstimatedCost(firstDayOfMonth, lastDayOfMonth, _estimatedCostLast)

	ec2Usage := <-_ec2UsageChan
	s3Usage := <-_s3UsageChan
	rdsUsage := <-_rdsUsageChan
	elasticacheUsage := <-_elasticacheUsageChan
	cloudFrontUsage := <-_cloudFrontUsageChan
	costCurrent := <-_estimatedCostCurrent
	costLast := <-_estimatedCostLast

	slackAttachments := make([]SlackAttachment, 0)

	// Add ec2 usage
	ec2UsageAttachment := SlackAttachment{
		Fallback: "EC2 Usage <!channel>",
		PreText:  "EC2 Usage <!channel>",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	ec2UsageKeys := getSortedKeySlice(ec2Usage)
	for _, key := range ec2UsageKeys {
		ec2UsageAttachment.Fields = append(ec2UsageAttachment.Fields, SlackAttachmentField{
			Title: key,
			Value: ec2Usage[key],
			Short: true,
		})
	}
	slackAttachments = append(slackAttachments, ec2UsageAttachment)

	// Add s3 usage
	s3UsageAttachment := SlackAttachment{
		Fallback: "S3 Usage",
		PreText:  "S3 Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	s3UsageKeys := getSortedKeySlice(s3Usage)
	for _, key := range s3UsageKeys {
		s3UsageAttachment.Fields = append(s3UsageAttachment.Fields, SlackAttachmentField{
			Title: key,
			Value: s3Usage[key],
			Short: true,
		})
	}
	slackAttachments = append(slackAttachments, s3UsageAttachment)

	// Add cloudfront usage
	cloudFrontUsageAttachment := SlackAttachment{
		Fallback: "CloudFront Usage",
		PreText:  "CloudFront Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	cloudFrontUsageKeys := getSortedKeySlice(cloudFrontUsage)
	for _, key := range cloudFrontUsageKeys {
		cloudFrontUsageAttachment.Fields = append(cloudFrontUsageAttachment.Fields, SlackAttachmentField{
			Title: key,
			Value: cloudFrontUsage[key],
			Short: true,
		})
	}
	slackAttachments = append(slackAttachments, cloudFrontUsageAttachment)

	// Add RDS usage
	rdsUsageAttachment := SlackAttachment{
		Fallback: "RDS Usage",
		PreText:  "RDS Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	rdsUsageKeys := getSortedKeySlice(rdsUsage)
	for _, key := range rdsUsageKeys {
		rdsUsageAttachment.Fields = append(rdsUsageAttachment.Fields, SlackAttachmentField{
			Title: key,
			Value: rdsUsage[key],
			Short: true,
		})
	}
	slackAttachments = append(slackAttachments, rdsUsageAttachment)

	// Add ElastiCache usage
	elasticacheUsageAttachment := SlackAttachment{
		Fallback: "Elasticache Usage",
		PreText:  "Elasticache Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	elasticacheUsageKeys := getSortedKeySlice(elasticacheUsage)
	for _, key := range elasticacheUsageKeys {
		elasticacheUsageAttachment.Fields = append(elasticacheUsageAttachment.Fields, SlackAttachmentField{
			Title: key,
			Value: elasticacheUsage[key],
			Short: true,
		})
	}
	slackAttachments = append(slackAttachments, elasticacheUsageAttachment)

	// Add cost estimation
	costEstimationAttachment := SlackAttachment{
		Fallback: "Estimated Cost",
		PreText:  "Estimated Cost",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	costEstimationAttachment.Fields = append(costEstimationAttachment.Fields, SlackAttachmentField{
		Title: "Current Month",
		Value: fmt.Sprintf("$%.02f USD", costCurrent),
		Short: true,
	})
	costEstimationAttachment.Fields = append(costEstimationAttachment.Fields, SlackAttachmentField{
		Title: "Last Month",
		Value: fmt.Sprintf("$%.02f USD", costLast),
		Short: true,
	})

	slackAttachments = append(slackAttachments, costEstimationAttachment)

	slackAttachmentsBytes, _ := json.Marshal(SlackAttachments{Attacments: slackAttachments})
	fmt.Println(string(slackAttachmentsBytes))

	// call slack webhook URL.
	payload := strings.NewReader(string(slackAttachmentsBytes))
	req, _ := http.NewRequest("POST", slackWebhookURL, payload)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("cache-control", "no-cache")
	res, _ := http.DefaultClient.Do(req)
	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(string(body))

	_msgSent <- true
}

func main() {

	_estimatedCostCurrent = make(chan float64)
	_estimatedCostLast = make(chan float64)
	_ec2UsageChan = make(chan map[string]string)
	_s3UsageChan = make(chan map[string]string)
	_cloudFrontUsageChan = make(chan map[string]string)
	_elasticacheUsageChan = make(chan map[string]string)
	_rdsUsageChan = make(chan map[string]string)

	_msgSent = make(chan bool)

	cron := cron.New()

	if runtime.GOOS == "darwin" {
		// For debug local, every 5 seconds
		cron.AddFunc("0/5 * * * * ?", messageSlack)
		// Read the slack webhook url.
		slackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	} else {
		// From Monday to Friday, 9:00am everyday
		cron.AddFunc("0 0 1 * * MON-FRI", messageSlack)
		slackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	}

	fmt.Println(slackWebhookURL)

	cron.Start()
	defer cron.Stop()

	_msgSent <- false

	holder := make(chan int)
	for {
		fmt.Println(<-holder)
	}
}
