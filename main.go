package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/WUMUXIAN/aws-slack-bot/jobs"
	"github.com/robfig/cron"
)

func runCronJob() {
	// Check slack webhook
	if os.Getenv("SLACK_WEBHOOK_URL") == "" {
		fmt.Println("Please specify the slack webhook URL")
		return
	}

	// Set regions
	regions := []string{}
	if os.Getenv("REGIONS") != "" {
		regions = strings.Split(os.Getenv("REGIONS"), ",")
	}
	if len(regions) == 0 {
		regions = []string{"us-east-1"}
	}

	// Get cron definition
	cronDefinition := "0 0 1 * * MON-FRI"
	if os.Getenv("CRON_DEFINITION") != "" {
		cronDefinition = os.Getenv("CRON_DEFINITION")
	}

	// Start the cron jobs and hold the process.
	cron := cron.New()
	// Job for the us region
	slackJob := jobs.NewSlackJob(
		regions,
		os.Getenv("SLACK_WEBHOOK_URL"),
	)
	err := cron.AddJob(cronDefinition, slackJob)
	if err != nil {
		fmt.Println("Failed to schedule cron job:", err.Error())
		return
	}

	cron.Start()
	defer cron.Stop()

	holder := make(chan int)
	<-holder
}

func main() {
	runCronJob()
}
