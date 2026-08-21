module example

go 1.23

replace github.com/dadav/stampede => ../

require (
	github.com/dadav/stampede v0.5.1
	github.com/go-chi/chi/v5 v5.3.2
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/goware/singleflight v0.2.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
)
