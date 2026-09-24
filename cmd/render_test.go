package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRenderTableDerivesColumnsAndAligns(t *testing.T) {
	rows := []json.RawMessage{
		json.RawMessage(`{"id":1,"name":"web1","status":null}`),
		json.RawMessage(`{"id":23,"name":"db-primary","status":"active"}`),
	}

	var out bytes.Buffer
	if err := renderTable(&out, rows); err != nil {
		t.Fatalf("renderTable: %v", err)
	}

	want := "ID  NAME        STATUS\n" +
		"1   web1        -\n" +
		"23  db-primary  active\n"
	if out.String() != want {
		t.Errorf("table =\n%q\nwant\n%q", out.String(), want)
	}
}

func TestRenderTableCapsAtSixColumns(t *testing.T) {
	rows := []json.RawMessage{
		json.RawMessage(`{"a":1,"b":2,"c":3,"d":4,"e":5,"f":6,"g":7,"h":8}`),
	}

	var out bytes.Buffer
	if err := renderTable(&out, rows); err != nil {
		t.Fatalf("renderTable: %v", err)
	}

	want := "A  B  C  D  E  F\n1  2  3  4  5  6\n"
	if out.String() != want {
		t.Errorf("table =\n%q\nwant\n%q", out.String(), want)
	}
}

func TestRenderTableEmpty(t *testing.T) {
	var out bytes.Buffer
	if err := renderTable(&out, nil); err != nil {
		t.Fatalf("renderTable: %v", err)
	}
	if out.String() != "No results.\n" {
		t.Errorf("table = %q, want No results.", out.String())
	}
}

func TestObjectKeysKeepsDocumentOrder(t *testing.T) {
	keys := objectKeys(json.RawMessage(`{"z":1,"a":{"nested":true},"m":"x"}`))

	want := []string{"z", "a", "m"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for index := range want {
		if keys[index] != want[index] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
}

func TestDisplayCellShapes(t *testing.T) {
	cases := map[string]struct {
		value any
		want  string
	}{
		"null":   {nil, "-"},
		"true":   {true, "true"},
		"bool":   {false, "false"},
		"number": {json.Number("42"), "42"},
		"array":  {[]any{"a", "b"}, `["a","b"]`},
	}
	for name, testCase := range cases {
		if got := displayCell(testCase.value); got != testCase.want {
			t.Errorf("%s: cell = %q, want %q", name, got, testCase.want)
		}
	}
}

func TestRenderDataFormats(t *testing.T) {
	var out bytes.Buffer
	if err := renderData(&out, json.RawMessage(`{"name":"web1"}`), "json"); err != nil {
		t.Fatalf("renderData json: %v", err)
	}
	if want := "{\n    \"name\": \"web1\"\n}\n"; out.String() != want {
		t.Errorf("json = %q, want %q", out.String(), want)
	}

	out.Reset()
	if err := renderData(&out, json.RawMessage(`[]`), "table"); err != nil {
		t.Fatalf("renderData table: %v", err)
	}
	if out.String() != "No results.\n" {
		t.Errorf("table = %q, want No results.", out.String())
	}
}
