package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const maxTableColumns = 6

// objectKeys lists the top level keys of a JSON object in document order.
// Decoding into a map would lose that order, and the table needs it.
func objectKeys(raw json.RawMessage) []string {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	first, err := decoder.Token()
	if err != nil {
		return nil
	}
	if delimiter, ok := first.(json.Delim); !ok || delimiter != '{' {
		return nil
	}

	var keys []string
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return keys
		}
		key, _ := keyToken.(string)
		keys = append(keys, key)

		var skip json.RawMessage
		if err := decoder.Decode(&skip); err != nil {
			return keys
		}
	}

	return keys
}

func rowCells(raw json.RawMessage, columns []string) []string {
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	_ = decoder.Decode(&object)

	cells := make([]string, len(columns))
	for index, column := range columns {
		cells[index] = displayCell(object[column])
	}

	return cells
}

func displayCell(value any) string {
	switch typed := value.(type) {
	case nil:
		return "-"
	case bool:
		return strconv.FormatBool(typed)
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "-"
		}
		return string(encoded)
	}
}

// renderTable writes rows with columns derived from the first row, capped at
// maxTableColumns, with two spaces between columns and the last column
// unpadded.
func renderTable(w io.Writer, rows []json.RawMessage) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "No results.")
		return err
	}

	columns := objectKeys(rows[0])
	if len(columns) > maxTableColumns {
		columns = columns[:maxTableColumns]
	}

	headers := make([]string, len(columns))
	widths := make([]int, len(columns))
	for index, column := range columns {
		headers[index] = strings.ToUpper(column)
		widths[index] = len(headers[index])
	}

	table := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells := rowCells(row, columns)
		for index, cell := range cells {
			if len(cell) > widths[index] {
				widths[index] = len(cell)
			}
		}
		table = append(table, cells)
	}

	last := len(columns) - 1
	writeRow := func(cells []string) error {
		for index, cell := range cells {
			if index == last {
				if _, err := fmt.Fprint(w, cell); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintf(w, "%-*s  ", widths[index], cell); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(w)
		return err
	}

	if err := writeRow(headers); err != nil {
		return err
	}
	for _, cells := range table {
		if err := writeRow(cells); err != nil {
			return err
		}
	}

	return nil
}

// renderData writes the response as indented json or as a table built from
// the document order of the response keys.
func renderData(w io.Writer, raw json.RawMessage, format string) error {
	trimmed := bytes.TrimSpace(raw)

	if format == "json" {
		var indented bytes.Buffer
		if err := json.Indent(&indented, trimmed, "", "    "); err != nil {
			return err
		}
		_, err := fmt.Fprintln(w, indented.String())
		return err
	}

	if len(trimmed) > 0 && trimmed[0] == '[' {
		var rows []json.RawMessage
		if err := json.Unmarshal(trimmed, &rows); err != nil {
			return err
		}
		return renderTable(w, rows)
	}

	if len(trimmed) > 0 && trimmed[0] == '{' {
		return renderTable(w, []json.RawMessage{trimmed})
	}

	var scalar any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&scalar); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, displayCell(scalar))
	return err
}
