package main

import (
	"github.com/doraemonkeys/WindSend-Relay/server/storage/acl/model"
	"gorm.io/gen"
)

// gorm gen configure
// Use 'go run main.go' to generate dao code.

//go:generate go run main.go
func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "../../storage/acl/query",
		// ModelPkgPath: "../../storage/acl/model",
		Mode: gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
	})

	// path := `relay.db`
	// db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	// if err != nil {
	// 	panic("failed to connect database")
	// }
	// g.UseDB(db)

	g.ApplyBasic(
		model.RelayStatistic{},
		model.KeyValue{},
	)

	// g.GenerateAllTable()

	g.Execute()
}
