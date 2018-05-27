package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"daill.de/go-chart"
)

func drawChart(res http.ResponseWriter, req *http.Request) {
	sbc := chart.BubbleChart {
		Title:      "Test Bubble Chart",
		TitleStyle: chart.StyleShow(),

		Background: chart.Style{
			Padding: chart.Box{
				Top: 40,
			},
		},
		BubbleScale: 2.7,
		Height:   500,
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show: true,
				StrokeWidth: 1,
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: true,
				StrokeWidth: 1,
			},
		},
		Bubbles: []chart.BubbleValue{
			{Value: chart.Value{Value: 2.55, Label: "Blue"}, YVal: 1.1, XVal: 1.1},
			{Value: chart.Value{Value: 1, Label: "Yewllo"}, YVal: 4.0, XVal: 2.0},
			{Value: chart.Value{Value: 4.2, Label: "Green"}, YVal: 5.0, XVal: 3.0},
			{Value: chart.Value{Value:  3.2, Label: "Red"}, YVal: 1.1, XVal: 1.1},
			{Value: chart.Value{Value: 5.5, Label: "Some"}, YVal: 1.5, XVal: 1.6},
		},
	}

	res.Header().Set("Content-Type", "image/png")
	err := sbc.Render(chart.PNG, res)
	if err != nil {
		fmt.Printf("Error rendering chart: %v\n", err)
	}
}

func port() string {
	if len(os.Getenv("PORT")) > 0 {
		return os.Getenv("PORT")
	}
	return "8080"
}

func main() {
	listenPort := fmt.Sprintf(":%s", port())
	fmt.Printf("Listening on %s\n", listenPort)
	http.HandleFunc("/", drawChart)
	log.Fatal(http.ListenAndServe(listenPort, nil))
}
