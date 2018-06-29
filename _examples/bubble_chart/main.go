package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"github.com/daill/go-chart"
)

func drawChart(res http.ResponseWriter, req *http.Request) {
	sbc := chart.BubbleChart {
		Title:      "Test Bubble Chart",
		TitleStyle: chart.StyleShow(),

		Background: chart.Style{
			Padding: chart.Box{
				Top: 100,
			},
		},
		BubbleScale: 2.7,
		Height:   300,
		XAxis: chart.XAxis{
			Ticks: []chart.Tick{{0, "0"},{1, "1"}, {2, "2"}, {3, "3"}, {4, "4"}, {5, "5"}, {6, ""}},
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
			{Value: chart.Value{Value: 6, Label: "Test User"}, YVal: 252759000000000, XVal: 1},
			{Value: chart.Value{Value: 0, Label: "Test User 1"}, YVal: 0, XVal: 0},
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
