package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"time"

	rt "transit_realtime"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/proto"
)

func main() {
	// bdfmEndpoint := "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm"
	alerts_endpoint := "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/camsys%2Fsubway-alerts"
	req, err := http.NewRequest("GET", alerts_endpoint, nil)
	if err != nil {
		fmt.Println(err)
	}

	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
	}

	apiKey := os.Getenv("MTA_API_KEY")
	if apiKey == "" {
		fmt.Println("API_KEY environment variable is not set.")
	}

	req.Header["x-api-key"] = []string{apiKey}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	msg := &rt.FeedMessage{}

	if err := proto.Unmarshal(body, msg); err != nil {
		fmt.Println("Failed to parse FeedMessage:", err)
	}

	fd, err := os.Open("template.html")
	if err != nil {
		fmt.Println(err)
	}

	r := bufio.NewReader(fd)
	bytes, err := io.ReadAll(r)
	if err != nil {
		fmt.Println(err)
	}

	tmpl, err := template.New("test").Parse(string(bytes))
	if err != nil {
		fmt.Println(err)
	}

	data := msg.GetEntity()
	handler := func(w http.ResponseWriter, r *http.Request) {
		if err := http.reqChecker(r.Body); err != nil {
            fmt.Println(err)
		}
		tmpl.Execute(w, data)
	}

	http.HandleFunc("/", handler)
	fmt.Println(http.ListenAndServe(":80", nil))
}

func printMessage(msg *rt.FeedMessage) {
	fmt.Println(msg.GetHeader())
	for _, entity := range msg.GetEntity() {
		alert := entity.GetAlert()
		timeRange := entity.GetAlert().GetActivePeriod()[0]
		current := uint64(time.Now().Unix())
		containsCurrentTime := timeRange.End != nil && *timeRange.Start < current && *timeRange.End > current

		if containsCurrentTime {
			localStartTime := time.Unix(int64(*timeRange.Start), 0)
			localEndTime := time.Unix(int64(*timeRange.End), 0)

			// "Alerts are always for trips, so entities are TripDescriptors"
			if alert.GetInformedEntity()[0].GetRouteId() == "" {
				fmt.Println("Couldn't parse train:", alert.GetInformedEntity())
			} else {
				fmt.Println("Active alert:", alert.GetHeaderText().GetTranslation()[0].GetText(),
					" from", localStartTime, "to", localEndTime)
				fmt.Println()
			}
		} else if timeRange.End == nil {
			fmt.Println("Indefinite alert:", alert.GetHeaderText().GetTranslation()[0].GetText())
		}
	}
}
