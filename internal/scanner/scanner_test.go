package scanner_test

import (
	"testing"

	"github.com/lazyvunit/lazy_vunit/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sampleJSON = []byte(`{
  "export_format_version": {"major": 1, "minor": 0, "patch": 0},
  "files": [],
  "tests": [
    {
      "name": "lib.tb_alu.test_add",
      "location": {"file_name": "/proj/src/alu/tb_alu.vhd", "offset": 0, "length": 0},
      "attributes": {}
    },
    {
      "name": "lib.tb_alu.test_subtract",
      "location": {"file_name": "/proj/src/alu/tb_alu.vhd", "offset": 0, "length": 0},
      "attributes": {}
    },
    {
      "name": "lib.tb_uart.test_baud",
      "location": {"file_name": "/proj/src/uart/tb_uart.vhd", "offset": 0, "length": 0},
      "attributes": {}
    }
  ]
}`)

func TestParseJSON_ParsesNames(t *testing.T) {
	entries, err := scanner.ParseJSON(sampleJSON)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, "lib.tb_alu.test_add", entries[0].Name)
	assert.Equal(t, "lib", entries[0].Library)
	assert.Equal(t, "tb_alu", entries[0].Bench)
	assert.Equal(t, "test_add", entries[0].TestCase)
}

func TestParseJSON_SetsDir(t *testing.T) {
	entries, err := scanner.ParseJSON(sampleJSON)
	require.NoError(t, err)
	assert.Equal(t, "/proj/src/alu", entries[0].Dir)
	assert.Equal(t, "/proj/src/uart", entries[2].Dir)
}

func TestParseJSON_EmptyTests(t *testing.T) {
	data := []byte(`{"export_format_version":{"major":1,"minor":0,"patch":0},"files":[],"tests":[]}`)
	entries, err := scanner.ParseJSON(data)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestParseJSON_MalformedJSON(t *testing.T) {
	_, err := scanner.ParseJSON([]byte(`not json`))
	assert.Error(t, err)
}
