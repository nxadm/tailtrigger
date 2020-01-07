package main

import (
	"testing"
)

func TestImportConfigFile(t *testing.T) {
	fileWatches, foundErrors := importConfigFile("t/tailtrigger.yaml")
	if foundErrors != nil {
		t.Errorf("Unexpected error found: %+v", foundErrors)
	}
	if fileWatches[0].RecDelimRxString != "^#" {
		t.Errorf("Expected record-delimiter '^#', got %s",
			fileWatches[0].RecDelimRxString)
	}

	fileWatches, foundErrors = importConfigFile("t/tailtrigger_errors.yaml")
	if foundErrors == nil {
		t.Errorf("Expected errors not found")
	}
}
