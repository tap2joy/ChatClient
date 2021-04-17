module github.com/tap2joy/ChatClient

go 1.14

require (
	github.com/tap2joy/Protocols v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.23.0
)

replace github.com/tap2joy/Protocols => ./proto
