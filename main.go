package main

import (
	"github.com/WUMUXIAN/aws-slack-bot/jobs"
	"github.com/robfig/cron"
)

func runCronJob() {
	// Start the cron jobs and hold the process.
	cron := cron.New()

	// Job for the us region
	slackJob := jobs.NewSlackJob(
		[]string{"ap-southeast-1"},
		"https://hooks.slack.com/services/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",
	)
	// os.Getenv("CRON_SCHEDULE")
	// every 5 seconds
	// "0/5 * * * * ?"
	// every 9:00am SGT, MON-FRI
	// "0 0 1 * * MON-FRI"

	cron.AddJob("0/5 * * * * ?", slackJob)

	cron.Start()
	defer cron.Stop()

	holder := make(chan int)
	<-holder
}

func main() {
	runCronJob()
}
