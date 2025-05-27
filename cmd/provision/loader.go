package main

import (
	"encoding/json"
	"os"

	"userclouds.com/infra/ucerr"
)

func loadFile(filename string, provData any) error {
	f, err := os.Open(filename)
	if err != nil {
		return ucerr.Errorf("failed to open provisioning file %v: %w", filename, err)
	}

	if err := json.NewDecoder(f).Decode(provData); err != nil {
		return ucerr.Errorf("couldn't decode provisioning file %v: %w", filename, err)
	}
	return nil
}
