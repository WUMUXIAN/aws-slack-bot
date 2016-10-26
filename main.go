package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

var estimatedCost chan float64
var runningInstances chan int

type result struct {
	Datapoints datapoints
}

func getEstimatedCost() {
	sess := session.New(&aws.Config{Region: aws.String("us-east-1")})

	svc := cloudwatch.New(sess)

	now := time.Now().UTC()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()
	firstDayOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

	fmt.Println("Start time: ", firstDayOfMonth)
	fmt.Println("End time: ", lastDayOfMonth)

	params := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/Billing"),
		StartTime:  aws.Time(firstDayOfMonth),
		EndTime:    aws.Time(lastDayOfMonth),
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

	estimatedCost <- result.Datapoints[0].Maximum
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
	go getEstimatedCost()

	var cost float64
	var count int

	for {
		cost = <-estimatedCost
		count = <-runningInstances
		fmt.Println("Estimated Cost This Month: ", cost)
		fmt.Println("Running Instance Count: ", count)

		url := "https://hooks.slack.com/services/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"

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
		               "title":"Estimated Cost Current Month",
		               "value":"$` + strconv.FormatFloat(cost, 'f', 2, 64) + ` USD",
		               "short":false
		            },
		         ]
		      }
		   ]
		}
		`)

		req, _ := http.NewRequest("POST", url, payload)

		req.Header.Add("content-type", "application/json")
		req.Header.Add("cache-control", "no-cache")

		res, _ := http.DefaultClient.Do(req)

		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)

		fmt.Println(string(body))
	}
}

func main() {

	estimatedCost = make(chan float64)
	runningInstances = make(chan int)

	cron := cron.New()

	if runtime.GOOS == "darwin" {
		// For debug local, every 5 seconds
		cron.AddFunc("0/5 * * * * ?", messageSlack)
	} else {
		// From Monday to Friday, 9:00am everyday
		cron.AddFunc("0 0 1 * * MON-FRI", messageSlack)
	}

	cron.Start()
	defer cron.Stop()

	holder := make(chan int)
	for {
		fmt.Println(<-holder)
	}
}
