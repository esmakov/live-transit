module github.com/esmakov/live-transit

go 1.20

require (
	google.golang.org/protobuf v1.31.0
	transit_realtime v0.0.0
)

replace transit_realtime v0.0.0 => ./transit_realtime
