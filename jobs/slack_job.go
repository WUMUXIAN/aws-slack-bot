// Package jobs defines a few cron jobs that can be scheduled to run.
package jobs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/WUMUXIAN/aws-slack-bot/stats"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
)

// RegionUsage represents the various resource usage of a region.
type RegionUsage struct {
	Sess                              *session.Session
	ec2UsageChan                      chan map[string]string
	s3UsageChan                       chan map[string]string
	cloudFrontUsageChan               chan map[string]string
	elasticacheUsageChan              chan map[string]string
	rdsUsageChan                      chan map[string]string
	billingEstimationCurrentMonthChan chan []float64
	billingEstimationLastMonthChan    chan []float64
}

// SlackJob defines a slack cron job
type SlackJob struct {
	regionUsage     map[string]RegionUsage
	slackWebhookURL string
}

// NewSlackJob creates a new slack cron job.
func NewSlackJob(regions []string, webhookURL string) SlackJob {
	slackJob := SlackJob{
		regionUsage:     make(map[string]RegionUsage),
		slackWebhookURL: webhookURL,
	}
	for _, region := range regions {
		sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
		slackJob.regionUsage[region] = RegionUsage{
			Sess:                              sess,
			ec2UsageChan:                      make(chan map[string]string),
			s3UsageChan:                       make(chan map[string]string),
			cloudFrontUsageChan:               make(chan map[string]string),
			elasticacheUsageChan:              make(chan map[string]string),
			rdsUsageChan:                      make(chan map[string]string),
			billingEstimationCurrentMonthChan: make(chan []float64),
			billingEstimationLastMonthChan:    make(chan []float64),
		}
	}
	return slackJob
}

func getSlackAttachmentFields(m map[string]string) []SlackAttachmentField {
	fields := make([]SlackAttachmentField, 0)
	keys := stats.GetSortedKeySlice(m)
	for _, key := range keys {
		fields = append(fields, SlackAttachmentField{
			Title: key,
			Value: m[key],
			Short: true,
		})
	}
	return fields
}

// Run runs the slack cron job.
func (o SlackJob) Run() {
	// Get EC2 usage for current session

	ec2UsageMap := make(map[string]map[string]string)
	s3UsageMap := make(map[string]map[string]string)
	cloudFrontUsageMap := make(map[string]map[string]string)
	rdsUsageMap := make(map[string]map[string]string)
	elasticacheUsageMap := make(map[string]map[string]string)
	billingEstimationCurrentMonth := []float64{0, 0}
	billingEstimationLastMonth := []float64{0, 0}

	parition := endpoints.AwsPartition()

	for region, usage := range o.regionUsage {
		go func() {
			usage.ec2UsageChan <- stats.GetEC2Usage(usage.Sess)
		}()

		now := time.Now().UTC()
		currentYear, currentMonth, _ := now.Date()
		currentLocation := now.Location()
		firstDayOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
		lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, 0).Add(-time.Second)
		go func() {
			usage.s3UsageChan <- stats.GetS3Usage(usage.Sess, firstDayOfMonth, lastDayOfMonth)
		}()
		go func() {
			usage.cloudFrontUsageChan <- stats.GetCloudFrontUsage(usage.Sess, firstDayOfMonth, lastDayOfMonth)
		}()
		go func() {
			usage.rdsUsageChan <- stats.GetRDSUsage(usage.Sess, firstDayOfMonth, lastDayOfMonth)
		}()
		go func() {
			usage.elasticacheUsageChan <- stats.GetElasticacheUsage(usage.Sess, firstDayOfMonth, lastDayOfMonth)
		}()
		go func() {
			// fmt.Println("Gather accumulated estimated billing for this month", firstDayOfMonth, lastDayOfMonth)
			month, average := stats.GetEstimatedBilling(usage.Sess, firstDayOfMonth, lastDayOfMonth)
			usage.billingEstimationCurrentMonthChan <- []float64{month, average}
		}()

		firstDayOfLastMonth := firstDayOfMonth.AddDate(0, -1, 0)
		lastDayOfLastMonth := firstDayOfLastMonth.AddDate(0, 1, 0).Add(-time.Second)
		go func() {
			// fmt.Println("Gather accumulated estimated billing for last month", firstDayOfLastMonth, lastDayOfLastMonth)
			month, average := stats.GetEstimatedBilling(usage.Sess, firstDayOfLastMonth, lastDayOfLastMonth)
			usage.billingEstimationLastMonthChan <- []float64{month, average}
		}()

		ec2UsageMap[region] = <-usage.ec2UsageChan
		s3UsageMap[region] = <-usage.s3UsageChan
		cloudFrontUsageMap[region] = <-usage.cloudFrontUsageChan
		rdsUsageMap[region] = <-usage.rdsUsageChan
		elasticacheUsageMap[region] = <-usage.elasticacheUsageChan

		billingEstimation := <-usage.billingEstimationCurrentMonthChan
		billingEstimationCurrentMonth[0] += billingEstimation[0]
		billingEstimationCurrentMonth[1] += billingEstimation[1]

		billingEstimation = <-usage.billingEstimationLastMonthChan
		billingEstimationLastMonth[0] += billingEstimation[0]
		billingEstimationLastMonth[1] += billingEstimation[1]
	}

	slackAttachments := make([]SlackAttachment, 0)

	// Add ec2 usage
	ec2UsageAttachment := SlackAttachment{
		Fallback: "EC2 Usage",
		PreText:  "EC2 Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	for region, ec2Usage := range ec2UsageMap {
		if len(ec2Usage) == 0 {
			continue
		}
		paritionRegion := parition.Regions()[region]
		ec2UsageAttachment.Fields = append(ec2UsageAttachment.Fields, SlackAttachmentField{
			Title: "",
			Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
			Short: false,
		})
		ec2UsageAttachment.Fields = append(ec2UsageAttachment.Fields, getSlackAttachmentFields(ec2Usage)...)
	}
	slackAttachments = append(slackAttachments, ec2UsageAttachment)

	// Add s3 usage
	s3UsageAttachment := SlackAttachment{
		Fallback: "S3 Usage",
		PreText:  "S3 Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	for region, s3Usage := range s3UsageMap {
		if len(s3Usage) == 0 {
			continue
		}
		paritionRegion := parition.Regions()[region]
		s3UsageAttachment.Fields = append(s3UsageAttachment.Fields, SlackAttachmentField{
			Title: "",
			Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
			Short: false,
		})
		s3UsageAttachment.Fields = append(s3UsageAttachment.Fields, getSlackAttachmentFields(s3Usage)...)
	}
	slackAttachments = append(slackAttachments, s3UsageAttachment)

	// Add cloudfront usage
	cloudFrontUsageAttachment := SlackAttachment{
		Fallback: "CloudFront Usage",
		PreText:  "CloudFront Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	for region, cloudFrontUsage := range cloudFrontUsageMap {
		if len(cloudFrontUsage) == 0 {
			continue
		}
		paritionRegion := parition.Regions()[region]
		cloudFrontUsageAttachment.Fields = append(cloudFrontUsageAttachment.Fields, SlackAttachmentField{
			Title: "",
			Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
			Short: false,
		})
		cloudFrontUsageAttachment.Fields = append(cloudFrontUsageAttachment.Fields, getSlackAttachmentFields(cloudFrontUsage)...)
	}
	slackAttachments = append(slackAttachments, cloudFrontUsageAttachment)

	// Add RDS usage
	rdsUsageAttachment := SlackAttachment{
		Fallback: "RDS Usage",
		PreText:  "RDS Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	for region, rdsUsage := range rdsUsageMap {
		if len(rdsUsage) == 0 {
			continue
		}
		paritionRegion := parition.Regions()[region]
		rdsUsageAttachment.Fields = append(rdsUsageAttachment.Fields, SlackAttachmentField{
			Title: "",
			Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
			Short: false,
		})
		rdsUsageAttachment.Fields = append(rdsUsageAttachment.Fields, getSlackAttachmentFields(rdsUsage)...)
	}

	slackAttachments = append(slackAttachments, rdsUsageAttachment)

	// Add ElastiCache usage
	elasticacheUsageAttachment := SlackAttachment{
		Fallback: "Elasticache Usage",
		PreText:  "Elasticache Usage",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	for region, elasticacheUsage := range elasticacheUsageMap {
		if len(elasticacheUsage) == 0 {
			continue
		}
		paritionRegion := parition.Regions()[region]
		elasticacheUsageAttachment.Fields = append(elasticacheUsageAttachment.Fields, SlackAttachmentField{
			Title: "",
			Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
			Short: false,
		})
		elasticacheUsageAttachment.Fields = append(elasticacheUsageAttachment.Fields, getSlackAttachmentFields(elasticacheUsage)...)
	}
	slackAttachments = append(slackAttachments, elasticacheUsageAttachment)

	// Add cost estimation
	billingEstimationAttachment := SlackAttachment{
		Fallback: "Estimated Billing",
		PreText:  "Estimated Billing",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	billingEstimationAttachment.Fields = append(billingEstimationAttachment.Fields, SlackAttachmentField{
		Title: "Daily Average This Month",
		Value: fmt.Sprintf("$%.02f USD", billingEstimationCurrentMonth[1]),
		Short: true,
	})
	billingEstimationAttachment.Fields = append(billingEstimationAttachment.Fields, SlackAttachmentField{
		Title: "Accumulated This Month",
		Value: fmt.Sprintf("$%.02f USD", billingEstimationCurrentMonth[0]),
		Short: true,
	})
	billingEstimationAttachment.Fields = append(billingEstimationAttachment.Fields, SlackAttachmentField{
		Title: "Daily Average Last Month",
		Value: fmt.Sprintf("$%.02f USD", billingEstimationLastMonth[1]),
		Short: true,
	})
	billingEstimationAttachment.Fields = append(billingEstimationAttachment.Fields, SlackAttachmentField{
		Title: "Accumulated Last Month",
		Value: fmt.Sprintf("$%.02f USD", billingEstimationLastMonth[0]),
		Short: true,
	})
	slackAttachments = append(slackAttachments, billingEstimationAttachment)

	slackAttachmentsBytes, _ := json.Marshal(SlackAttachments{Attacments: slackAttachments})
	// fmt.Println(string(slackAttachmentsBytes))

	// call slack webhook URL.
	fmt.Println("Sending report to slack channel")
	payload := strings.NewReader(string(slackAttachmentsBytes))
	req, _ := http.NewRequest("POST", o.slackWebhookURL, payload)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("cache-control", "no-cache")
	res, _ := http.DefaultClient.Do(req)
	body, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	fmt.Println(string(body))
}

// SlackAttachmentField defines a slack attachment field
type SlackAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackAttachment defines a slack attachment
type SlackAttachment struct {
	Fallback   string                 `json:"fallback,omitempty"`
	Color      string                 `json:"color,omitempty"`
	PreText    string                 `json:"pretext,omitempty"`
	AuthorName string                 `json:"author_name,omitempty"`
	AuthorLink string                 `json:"author_link,omitempty"`
	AuthorIcon string                 `json:"author_icon,omitempty"`
	Title      string                 `json:"title,omitempty"`
	TitleLink  string                 `json:"title_link,omitempty"`
	Text       string                 `json:"text,omitempty"`
	Fields     []SlackAttachmentField `json:"fields,omitempty"`
	ImageURL   string                 `json:"image_url,omitempty"`
	ThumbURL   string                 `json:"thumb_url,omitempty"`
	Footer     string                 `json:"footer,omitempty"`
	FooterIcon string                 `json:"footer_icon,omitempty"`
	TS         int64                  `json:"ts,omitempty"`
}

// SlackAttachments defines slack attachments
type SlackAttachments struct {
	Attacments []SlackAttachment `json:"attachments"`
}
