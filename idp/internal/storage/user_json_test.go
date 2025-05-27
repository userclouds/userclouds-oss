package storage

// to run
//   go test -bench=. ./idp/internal/storage -run Benchmark

import (
	"encoding/json"
	"testing"

	gojson "github.com/goccy/go-json"
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	easyjson "github.com/userclouds/easyjson"

	"userclouds.com/idp/userstore"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucjson"
)

func setup() []byte {
	u := uuid.Must(uuid.FromString("b60d3e57-f8cc-484a-84b9-e2f0056093a2"))
	user := &User{
		BaseUser: BaseUser{
			VersionBaseModel: ucdb.NewVersionBaseWithID(u),
			OrganizationID:   u,
			Region:           region.DataRegion("us-east-1"),
		},
		Profile: userstore.Record{
			"name": "John Doe",
		},
	}

	j, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}

	return j
}
func UnmarshalNativeJSON(data []byte, v *User) error {
	return json.Unmarshal(data, v)
}

func BenchmarkNativeJSON(b *testing.B) {
	jsonData := setup()
	for b.Loop() {
		var u User
		if err := UnmarshalNativeJSON(jsonData, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoJSON(b *testing.B) {
	jsonData := setup()
	for b.Loop() {
		var u User
		if err := UnmarshalGoJSON(jsonData, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func UnmarshalGoJSON(data []byte, v *User) error {
	// example alternative (could be ffjson, easyjson, etc.)
	return gojson.Unmarshal(data, v) // replace with real alt
}

func BenchmarkJSONIter(b *testing.B) {
	jsonData := setup()
	for b.Loop() {
		var u User
		if err := UnmarshalJSONIter(jsonData, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func UnmarshalJSONIter(data []byte, v *User) error {
	return jsoniter.Unmarshal(data, v)
}

func BenchmarkEasyJSON(b *testing.B) {
	jsonData := setup()
	for b.Loop() {
		var u User
		if err := UnmarshalEasyJSON(jsonData, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func UnmarshalEasyJSON(data []byte, v *User) error {
	return easyjson.Unmarshal(data, v)
}

func BenchmarkUCJSON(b *testing.B) {
	jsonData := setup()
	for b.Loop() {
		var u User
		if err := UnmarshalUCJSON(jsonData, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func UnmarshalUCJSON(data []byte, v *User) error {
	return ucjson.Unmarshal(data, v)
}
