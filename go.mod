module github.com/esmakov/live-transit

go 1.20

require (
	google.golang.org/protobuf v1.31.0
	transit_realtime v0.0.0
)

require github.com/joho/godotenv v1.5.1 // indirect

replace transit_realtime v0.0.0 => ./transit_realtime
