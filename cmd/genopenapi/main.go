package main

import (
	"context"

	"userclouds.com/tools/generate/genopenapi"
)

func main() {
	ctx := context.Background()
	genopenapi.Run(ctx)
}
