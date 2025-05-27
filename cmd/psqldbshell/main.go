package main

import (
	"context"
	"fmt"
	"os"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/dbdata"
)

func main() {
	ctx := context.Background()
	args := os.Args[1:]
	if len(args) == 0 {
		panic("No DB name provided")

	} else if len(args) > 1 {
		panic("Too many arguments provided")
	}
	dbName := args[0]
	svcData, err := dbdata.GetDatabaseData(ctx, dbName)
	if err != nil {
		panic(err)
	}
	_, masterURL, err := ucdb.GetPostgresURLs(ctx, false, svcData.DBCfg, universe.Current(), region.Current())
	if err != nil {
		panic(err)
	}
	fmt.Println(masterURL)
}
