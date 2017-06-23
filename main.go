package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/robfig/cron"
)

type datapoint struct {
	Maximum   float64
	Timestamp string
	Unit      string
}

type datapoints []datapoint

func (d datapoints) Len() int {
	return len(d)
}

func (d datapoints) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d datapoints) Less(i, j int) bool {
	iTime, _ := time.Parse(time.RFC3339, d[i].Timestamp)
	jTime, _ := time.Parse(time.RFC3339, d[j].Timestamp)
	return iTime.After(jTime)
}

var estimatedCostCurrent chan float64
var estimatedCostLast chan float64

var runningInstances chan int
var slackWebhookURL string

type result struct {
	Datapoints datapoints
}

func getEstimatedCost(startTime, endTime time.Time, estimatedCost chan<- float64) {
	sess := session.New(&aws.Config{Region: aws.String("us-east-1")})

	svc := cloudwatch.New(sess)

	fmt.Println("Start time: ", startTime)
	fmt.Println("End time: ", endTime)

	params := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/Billing"),
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		MetricName: aws.String("EstimatedCharges"),
		Period:     aws.Int64(86400),
		Statistics: []*string{
			aws.String("Maximum"),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("Currency"),
				Value: aws.String("USD"),
			},
		},
	}

	resp, err := svc.GetMetricStatistics(params)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	jsonBody, _ := json.Marshal(resp)

	var result result
	json.Unmarshal(jsonBody, &result)
	sort.Sort(result.Datapoints)

	if len(result.Datapoints) > 0 {
		estimatedCost <- result.Datapoints[0].Maximum
	} else {
		estimatedCost <- 0
	}
}

func getRunningInstanceCount() {
	sess := session.New(&aws.Config{Region: aws.String("us-east-1")})

	svc := ec2.New(sess)

	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
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
		return
	}
	count := 0
	for i := 0; i < len(resp.Reservations); i++ {
		count += len(resp.Reservations[i].Instances)
	}

	runningInstances <- count
}

func messageSlack() {

	go getRunningInstanceCount()

	now := time.Now().UTC()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()
	firstDayOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

	go getEstimatedCost(firstDayOfMonth, lastDayOfMonth, estimatedCostCurrent)

	firstDayOfMonth = firstDayOfMonth.AddDate(0, -1, 0)
	lastDayOfMonth = firstDayOfMonth.AddDate(0, 1, -1)

	go getEstimatedCost(firstDayOfMonth, lastDayOfMonth, estimatedCostLast)

	var costCurrent float64
	var costLast float64
	var count int

	for {
		costCurrent = <-estimatedCostCurrent
		costLast = <-estimatedCostLast
		count = <-runningInstances
		// fmt.Println("Estimated Cost This Month: ", cost)
		// fmt.Println("Running Instance Count: ", count)

		payload := strings.NewReader(`
		{
		   "attachments":[
		      {
		         "fallback":"AWS Usage Report",
		         "pretext":"AWS Usage Report",
		         "color":"#D00000",
		         "fields":[
		            {
		               "title":"Running Instances",
		               "value":"` + strconv.Itoa(count) + `",
		               "short":false
		            },
		            {
		               "title":"Estimated Cost Last Month",
		               "value":"$` + strconv.FormatFloat(costLast, 'f', 2, 64) + ` USD",
		               "short":false
		            },
		            {
		               "title":"Estimated Cost Current Month",
		               "value":"$` + strconv.FormatFloat(costCurrent, 'f', 2, 64) + ` USD",
		               "short":false
		            },		            
		         ]
		      }
		   ]
		}
		`)

		req, _ := http.NewRequest("POST", slackWebhookURL, payload)

		req.Header.Add("content-type", "application/json")
		req.Header.Add("cache-control", "no-cache")

		res, _ := http.DefaultClient.Do(req)

		body, _ := ioutil.ReadAll(res.Body)

		res.Body.Close()

		fmt.Println(string(body))
	}
}

func main() {

	estimatedCostCurrent = make(chan float64)
	estimatedCostLast = make(chan float64)
	runningInstances = make(chan int)

	cron := cron.New()

	if runtime.GOOS == "darwin" {
		// For debug local, every 5 seconds
		cron.AddFunc("0/5 * * * * ?", messageSlack)
		// Read the slack webhook url.
		bytes, _ := ioutil.ReadFile("slack_webhook_url")
		slackWebhookURL = string(bytes)
	} else {
		// From Monday to Friday, 9:00am everyday
		cron.AddFunc("0 0 1 * * MON-FRI", messageSlack)
		homeDir := os.Getenv("HOME")
		bytes, _ := ioutil.ReadFile(filepath.Join(homeDir, "slack_webhook_url"))
		slackWebhookURL = string(bytes)
	}

	// fmt.Println(slackWebhookURL)

	cron.Start()
	defer cron.Stop()

	holder := make(chan int)
	for {
		fmt.Println(<-holder)
	}
}
