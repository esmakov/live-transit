package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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
		log.Fatalln(err)
	}

	if err := godotenv.Load(); err != nil {
		log.Fatalln("Couldn't load .env")
	}
	apiKey := os.Getenv("MTA_API_KEY")
	if apiKey == "" {
		log.Fatalln("API_KEY environment variable is not set.")
	}

	req.Header["x-api-key"] = []string{apiKey}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	msg := &rt.FeedMessage{}

	if err := proto.Unmarshal(body, msg); err != nil {
		log.Fatalln("Failed to parse FeedMessage:", err)
	}

	stationMap := makeStationMap()

	printMessage(msg, stationMap)
}

func makeStationMap() map[string]string {
	file, err := os.Open("stationmappings.txt")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	sm := make(map[string]string)

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		header := scanner.Text()
		fmt.Println(header)
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			fmt.Println("Error: Insufficient columns in the line")
			continue
		}
		gtfsStopID := strings.Trim(fields[0], `"`)
		stopName := strings.Trim(fields[1], `"`)
		sm[gtfsStopID] = stopName
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("Could not scan line")
	}

	return sm
}

func printMessage(msg *rt.FeedMessage, stationMap map[string]string) {
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
			affectedRoutes := alert.GetInformedEntity()
			if alert.GetInformedEntity()[0].GetRouteId() == "" {
				fmt.Println("Couldn't parse train:", alert.GetInformedEntity())
			} else {
				fmt.Println("Active alert:", alert.GetHeaderText().GetTranslation()[0].GetText(),
					" from", localStartTime, "to", localEndTime)
				fmt.Println("\tTrip description:")
				for i, v := range affectedRoutes {
					if route := v.GetRouteId(); route != "" {
						fmt.Print("\tAffecting the ", route, " train at station(s): ")
					} else if stop := v.GetStopId(); stop != "" {
						if v, ok := stationMap[stop]; ok {
							fmt.Print(v)
						} else {
							fmt.Print(stop)
						}

						if i < len(affectedRoutes)-1 {
							fmt.Print(", ")
						}
					}
				}
				fmt.Println()
			}
		} else if timeRange.End == nil {
			fmt.Println("Indefinite alert:", alert.GetHeaderText().GetTranslation()[0].GetText())
		}
	}
}
