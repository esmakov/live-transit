package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	rt "transit_realtime"

	"google.golang.org/protobuf/proto"
)

func main() {
	// go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	// bdfmEndpoint := "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm"
	const alerts_endpoint = "https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/camsys%2Fsubway-alerts"
	client := &http.Client{}

	tmpl := template.Must(template.ParseFiles("template.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method)

		resp, err := client.Get(alerts_endpoint)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		msg := &rt.FeedMessage{}

		if err := proto.Unmarshal(body, msg); err != nil {
			log.Fatal("Failed to parse FeedMessage:", err)
		}

		tmpl.Execute(w, struct {
			AlertGroups []alertGroup
		}{processResponse(msg)})
	})

	http.ListenAndServe(":80", nil)
}

type alertGroup struct {
	RouteID string
	Alerts  []alertBody
}

type alertBody struct {
	Summary    string
	TimeStart  string
	TimeEnd    string
	HasStarted bool
}

func processResponse(msg *rt.FeedMessage) []alertGroup {
	var groups []alertGroup
	m := make(map[string][]alertBody, 10)

	for _, entity := range msg.GetEntity() {
		alert := entity.Alert
		timeRange := alert.ActivePeriod[0]
		now := uint64(time.Now().Unix())

		if timeRange.End != nil && *timeRange.End < now {
			continue
		}
		// if timeRange.Start == nil || timeRange.End == nil || *timeRange.End < now {
		// 	continue
		// }

		// "Alerts are always for trips, so entities are TripDescriptors"
		if alert.InformedEntity[0].RouteId == nil {
			var s string
			alert.InformedEntity[0].RouteId = &s
		}

		RouteID := *alert.InformedEntity[0].RouteId

		localStartTime := time.Unix(int64(*timeRange.Start), 0)
		var localEndTime time.Time
		if timeRange.End != nil {
			localEndTime = time.Unix(int64(*timeRange.End), 0)
		}

		m[RouteID] = append(m[RouteID], alertBody{
			Summary:    *alert.HeaderText.Translation[0].Text,
			TimeStart:  localStartTime.String(),
			TimeEnd:    localEndTime.String(),
			HasStarted: localStartTime.Before(time.Now()),
		})
	}

	for k, v := range m {
		groups = append(groups, alertGroup{k, v})
	}

	slices.SortFunc(groups, func(a alertGroup, b alertGroup) int { return strings.Compare(a.RouteID, b.RouteID) })
	return groups
}
