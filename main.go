package main

import (
	"os"

	"github.com/WUMUXIAN/aws-slack-bot/jobs"
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

var sess *session.Session

func init() {
	sess = session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
}

func messageSlack() {
	_estimatedCostCurrent = make(chan float64)
	_estimatedCostLast = make(chan float64)
	_ec2UsageChan = make(chan map[string]string)
	_s3UsageChan = make(chan map[string]string)
	_cloudFrontUsageChan = make(chan map[string]string)
	_elasticacheUsageChan = make(chan map[string]string)
	_rdsUsageChan = make(chan map[string]string)
	// _msgSent = make(chan bool)

	// cron := cron.New()
	//
	// if runtime.GOOS == "darwin" {
	// 	// For debug local, every 5 seconds
	// 	cron.AddFunc("0/5 * * * * ?", messageSlack)
	// 	// Read the slack webhook url.
	// 	slackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	// } else {
	// 	// From Monday to Friday, 9:00am everyday
	// 	cron.AddFunc("0 0 1 * * MON-FRI", messageSlack)
	// 	slackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	// }

	// fmt.Println(slackWebhookURL)
}

func runCronJob() {
	// Start the cron jobs and hold the process.
	cron := cron.New()

	// Job for the us region
	slackJob := jobs.SlackJob{
		Sess:            session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")})),
		SlackWebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
	}
	cron.AddJob(os.Getenv("CRON_SCHEDULE"), slackJob)

	cron.Start()
	defer cron.Stop()

	holder := make(chan int)
	<-holder
}

func main() {
	runCronJob()
}
