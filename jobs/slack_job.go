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
	"github.com/aws/aws-sdk-go/aws/session"
)

// RegionUsage represents the various resource usage of a region.
type RegionUsage struct {
	Sess                 *session.Session
	ec2UsageChan         chan map[string]string
	s3UsageChan          chan map[string]string
	cloudFrontUsageChan  chan map[string]string
	elasticacheUsageChan chan map[string]string
	rdsUsageChan         chan map[string]string
}

// SlackJob defines a slack cron job
type SlackJob struct {
	regionUsage     map[string]RegionUsage
	costCurrentChan chan float64
	costLastChan    chan float64
	slackWebhookURL string
}

func NewSlackJob(regions []string, webhookURL string) SlackJob {
	slackJob := SlackJob{
		regionUsage:     make(map[string]RegionUsage),
		slackWebhookURL: webhookURL,
		costCurrentChan: make(chan float64),
		costLastChan:    make(chan float64),
	}
	for _, region := range regions {
		sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
		slackJob.regionUsage[region] = RegionUsage{
			Sess:                 sess,
			ec2UsageChan:         make(chan map[string]string),
			s3UsageChan:          make(chan map[string]string),
			cloudFrontUsageChan:  make(chan map[string]string),
			elasticacheUsageChan: make(chan map[string]string),
			rdsUsageChan:         make(chan map[string]string),
		}
	}
	return slackJob
}

// Run runs the slack cron job.
func (o SlackJob) Run() {
	// Get EC2 usage for current session
	for _, usage := range o.regionUsage {
		go func() {
			usage.ec2UsageChan <- stats.GetEC2Usage(usage.Sess)
		}()

		now := time.Now().UTC()
		currentYear, currentMonth, _ := now.Date()
		currentLocation := now.Location()
		firstDayOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
		lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)
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
			o.costCurrentChan <- stats.GetEstimatedCost(usage.Sess, firstDayOfMonth, lastDayOfMonth)
		}()

		firstDayOfMonth = firstDayOfMonth.AddDate(0, -1, 0)
		lastDayOfMonth = firstDayOfMonth.AddDate(0, 1, -1)
		go func() {
			o.costLastChan <- stats.GetEstimatedCost(usage.Sess, firstDayOfMonth, lastDayOfMonth)
		}()

		ec2Usage := <-usage.ec2UsageChan
		s3Usage := <-usage.s3UsageChan
		rdsUsage := <-usage.rdsUsageChan
		elasticacheUsage := <-usage.elasticacheUsageChan
		cloudFrontUsage := <-usage.cloudFrontUsageChan
		costCurrent := <-o.costCurrentChan
		costLast := <-o.costLastChan

		slackAttachments := make([]SlackAttachment, 0)

		// regionTitle := SlackAttachment{
		// 	Fallback: "EC2 Usage <!channel>",
		// 	PreText:  "EC2 Usage <!channel>",
		// 	Color:    "#D00000",
		// 	Fields:   make([]SlackAttachmentField, 0),
		// }

		// Add ec2 usage
		ec2UsageAttachment := SlackAttachment{
			Fallback: "EC2 Usage <!channel>",
			PreText:  "EC2 Usage <!channel>",
			Color:    "#D00000",
			Fields:   make([]SlackAttachmentField, 0),
		}
		ec2UsageKeys := stats.GetSortedKeySlice(ec2Usage)
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
		s3UsageKeys := stats.GetSortedKeySlice(s3Usage)
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
		cloudFrontUsageKeys := stats.GetSortedKeySlice(cloudFrontUsage)
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
		rdsUsageKeys := stats.GetSortedKeySlice(rdsUsage)
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
		elasticacheUsageKeys := stats.GetSortedKeySlice(elasticacheUsage)
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
		req, _ := http.NewRequest("POST", o.slackWebhookURL, payload)
		req.Header.Add("content-type", "application/json")
		req.Header.Add("cache-control", "no-cache")
		res, _ := http.DefaultClient.Do(req)
		body, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		fmt.Println(string(body))
	}
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
