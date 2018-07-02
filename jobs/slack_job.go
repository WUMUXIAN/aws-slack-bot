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
	billingEstimationCurrentMonthChan chan float64
	billingEstimationYesterdayChan    chan float64
	billingEstimationLastMonthChan    chan float64
}

// SlackJob defines a slack cron job
type SlackJob struct {
	regionUsage     map[string]RegionUsage
	slackWebhookURL string
}

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
			billingEstimationCurrentMonthChan: make(chan float64),
			billingEstimationYesterdayChan:    make(chan float64),
			billingEstimationLastMonthChan:    make(chan float64),
		}
	}
	return slackJob
}

// Run runs the slack cron job.
func (o SlackJob) Run() {
	// Get EC2 usage for current session

	ec2UsageMap := make(map[string]map[string]string)
	s3UsageMap := make(map[string]map[string]string)
	cloudFrontUsageMap := make(map[string]map[string]string)
	rdsUsageMap := make(map[string]map[string]string)
	elasticacheUsageMap := make(map[string]map[string]string)
	billingEstimationCurrentMonth := float64(0)
	billingEstimationYesterday := float64(0)
	billingEstimationLastMonth := float64(0)

	parition := endpoints.AwsPartition()

	for region, usage := range o.regionUsage {
		go func() {
			usage.ec2UsageChan <- stats.GetEC2Usage(usage.Sess)
		}()

		now := time.Now().UTC()
		currentYear, currentMonth, currentDay := now.Date()
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
			month, _ := stats.GetEstimatedBilling(usage.Sess, firstDayOfMonth, lastDayOfMonth)
			usage.billingEstimationCurrentMonthChan <- month
		}()

		firstDayOfLastMonth := firstDayOfMonth.AddDate(0, -1, 0)
		lastDayOfLastMonth := firstDayOfLastMonth.AddDate(0, 1, 0).Add(-time.Second)
		go func() {
			month, _ := stats.GetEstimatedBilling(usage.Sess, firstDayOfLastMonth, lastDayOfLastMonth)
			usage.billingEstimationLastMonthChan <- month
		}()

		endOfToday := time.Date(currentYear, currentMonth, currentDay, 0, 0, 0, 0, currentLocation).AddDate(0, 0, 1).Add(-time.Second)
		startOfYesterday := time.Date(currentYear, currentMonth, currentDay, 0, 0, 0, 0, currentLocation).AddDate(0, 0, -1)
		go func() {
			_, day := stats.GetEstimatedBilling(usage.Sess, startOfYesterday, endOfToday)
			usage.billingEstimationYesterdayChan <- day
		}()

		ec2Usage := <-usage.ec2UsageChan
		if len(ec2Usage) > 0 {
			ec2UsageMap[region] = ec2Usage
		}

		s3Usage := <-usage.s3UsageChan
		if len(s3Usage) > 0 {
			s3UsageMap[region] = s3Usage
		}

		cloudFrontUsage := <-usage.cloudFrontUsageChan
		if len(cloudFrontUsage) > 0 {
			cloudFrontUsageMap[region] = cloudFrontUsage
		}

		rdsUsage := <-usage.rdsUsageChan
		if len(rdsUsage) > 0 {
			rdsUsageMap[region] = rdsUsage
		}

		elasticacheUsage := <-usage.elasticacheUsageChan
		if len(elasticacheUsage) > 0 {
			elasticacheUsageMap[region] = elasticacheUsage
		}

		billingEstimationCurrentMonth += <-usage.billingEstimationCurrentMonthChan
		billingEstimationLastMonth += <-usage.billingEstimationLastMonthChan
		billingEstimationYesterday += <-usage.billingEstimationYesterdayChan
	}

	slackAttachments := make([]SlackAttachment, 0)

	// Add ec2 usage
	if len(ec2UsageMap) > 0 {
		ec2UsageAttachment := SlackAttachment{
			Fallback: "EC2 Usage",
			PreText:  "EC2 Usage",
			Color:    "#D00000",
			Fields:   make([]SlackAttachmentField, 0),
		}
		for region, ec2Usage := range ec2UsageMap {
			paritionRegion := parition.Regions()[region]
			ec2UsageAttachment.Fields = append(ec2UsageAttachment.Fields, SlackAttachmentField{
				Title: "",
				Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
				Short: false,
			})
			ec2UsageKeys := stats.GetSortedKeySlice(ec2Usage)
			for _, key := range ec2UsageKeys {
				ec2UsageAttachment.Fields = append(ec2UsageAttachment.Fields, SlackAttachmentField{
					Title: key,
					Value: ec2Usage[key],
					Short: true,
				})
			}
		}
		slackAttachments = append(slackAttachments, ec2UsageAttachment)
	}

	// Add s3 usage
	if len(s3UsageMap) > 0 {
		s3UsageAttachment := SlackAttachment{
			Fallback: "S3 Usage",
			PreText:  "S3 Usage",
			Color:    "#D00000",
			Fields:   make([]SlackAttachmentField, 0),
		}
		for region, s3Usage := range s3UsageMap {
			paritionRegion := parition.Regions()[region]
			s3UsageAttachment.Fields = append(s3UsageAttachment.Fields, SlackAttachmentField{
				Title: "",
				Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
				Short: false,
			})
			s3UsageKeys := stats.GetSortedKeySlice(s3Usage)
			for _, key := range s3UsageKeys {
				s3UsageAttachment.Fields = append(s3UsageAttachment.Fields, SlackAttachmentField{
					Title: key,
					Value: s3Usage[key],
					Short: true,
				})
			}
		}
		slackAttachments = append(slackAttachments, s3UsageAttachment)
	}

	// Add cloudfront usage
	if len(cloudFrontUsageMap) > 0 {
		cloudFrontUsageAttachment := SlackAttachment{
			Fallback: "CloudFront Usage",
			PreText:  "CloudFront Usage",
			Color:    "#D00000",
			Fields:   make([]SlackAttachmentField, 0),
		}
		for region, cloudFrontUsage := range cloudFrontUsageMap {
			paritionRegion := parition.Regions()[region]
			cloudFrontUsageAttachment.Fields = append(cloudFrontUsageAttachment.Fields, SlackAttachmentField{
				Title: "",
				Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
				Short: false,
			})
			cloudFrontUsageKeys := stats.GetSortedKeySlice(cloudFrontUsage)
			for _, key := range cloudFrontUsageKeys {
				cloudFrontUsageAttachment.Fields = append(cloudFrontUsageAttachment.Fields, SlackAttachmentField{
					Title: key,
					Value: cloudFrontUsage[key],
					Short: true,
				})
			}
		}
		slackAttachments = append(slackAttachments, cloudFrontUsageAttachment)
	}

	// Add RDS usage
	if len(rdsUsageMap) > 0 {
		rdsUsageAttachment := SlackAttachment{
			Fallback: "RDS Usage",
			PreText:  "RDS Usage",
			Color:    "#D00000",
			Fields:   make([]SlackAttachmentField, 0),
		}
		for region, rdsUsage := range rdsUsageMap {
			paritionRegion := parition.Regions()[region]
			rdsUsageAttachment.Fields = append(rdsUsageAttachment.Fields, SlackAttachmentField{
				Title: "",
				Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
				Short: false,
			})
			rdsUsageKeys := stats.GetSortedKeySlice(rdsUsage)
			for _, key := range rdsUsageKeys {
				rdsUsageAttachment.Fields = append(rdsUsageAttachment.Fields, SlackAttachmentField{
					Title: key,
					Value: rdsUsage[key],
					Short: true,
				})
			}
		}

		slackAttachments = append(slackAttachments, rdsUsageAttachment)
	}

	// Add ElastiCache usage
	if len(elasticacheUsageMap) > 0 {
		elasticacheUsageAttachment := SlackAttachment{
			Fallback: "Elasticache Usage",
			PreText:  "Elasticache Usage",
			Color:    "#D00000",
			Fields:   make([]SlackAttachmentField, 0),
		}
		for region, elasticacheUsage := range elasticacheUsageMap {
			paritionRegion := parition.Regions()[region]
			elasticacheUsageAttachment.Fields = append(elasticacheUsageAttachment.Fields, SlackAttachmentField{
				Title: "",
				Value: fmt.Sprintf("_&lt;%s: %s&gt;_", paritionRegion.Description(), region),
				Short: false,
			})
			elasticacheUsageKeys := stats.GetSortedKeySlice(elasticacheUsage)
			for _, key := range elasticacheUsageKeys {
				elasticacheUsageAttachment.Fields = append(elasticacheUsageAttachment.Fields, SlackAttachmentField{
					Title: key,
					Value: elasticacheUsage[key],
					Short: true,
				})
			}
		}

		slackAttachments = append(slackAttachments, elasticacheUsageAttachment)
	}

	// Add cost estimation
	billingEstimationAttachment := SlackAttachment{
		Fallback: "Estimated Billing",
		PreText:  "Estimated Billing",
		Color:    "#D00000",
		Fields:   make([]SlackAttachmentField, 0),
	}
	billingEstimationAttachment.Fields = append(billingEstimationAttachment.Fields, SlackAttachmentField{
		Title: "For Yesterday",
		Value: fmt.Sprintf("$%.02f USD", billingEstimationYesterday),
		Short: false,
	})
	billingEstimationAttachment.Fields = append(billingEstimationAttachment.Fields, SlackAttachmentField{
		Title: "Accumulated This Month",
		Value: fmt.Sprintf("$%.02f USD", billingEstimationCurrentMonth),
		Short: true,
	})
	billingEstimationAttachment.Fields = append(billingEstimationAttachment.Fields, SlackAttachmentField{
		Title: "Total Last Month",
		Value: fmt.Sprintf("$%.02f USD", billingEstimationLastMonth),
		Short: true,
	})
	slackAttachments = append(slackAttachments, billingEstimationAttachment)

	slackAttachmentsBytes, _ := json.Marshal(SlackAttachments{Attacments: slackAttachments})
	fmt.Println(string(slackAttachmentsBytes))

	// call slack webhook URL.
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
