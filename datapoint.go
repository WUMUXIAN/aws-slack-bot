package main

import "time"

type datapoint struct {
	Maximum     float64
	Minimum     float64
	SampleCount float64
	Sum         float64
	Average     float64
	Timestamp   string
	Unit        string
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
