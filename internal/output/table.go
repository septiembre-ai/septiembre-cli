package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

// writeTable renders v as a human-readable ASCII table using text/tabwriter.
// v must marshal to a JSON array of objects; if it does not, writeTable falls
// back to compact JSON so the output is always readable.
func writeTable(w io.Writer, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// Try to parse as an array of maps.
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		// Not a homogeneous array — fall back to indented JSON.
		return writeJSON(w, v)
	}

	// Collect and sort column headers for deterministic output.
	headerSet := make(map[string]struct{})
	for _, row := range rows {
		for k := range row {
			headerSet[k] = struct{}{}
		}
	}
	headers := make([]string, 0, len(headerSet))
	for k := range headerSet {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	// Print header row.
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(tw, "\t")
		}
		fmt.Fprint(tw, h)
	}
	fmt.Fprintln(tw)

	// Print data rows.
	for _, row := range rows {
		for i, h := range headers {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			val := row[h]
			switch v := val.(type) {
			case string:
				fmt.Fprint(tw, v)
			case nil:
				fmt.Fprint(tw, "")
			default:
				// Inline JSON for nested structures.
				b, _ := json.Marshal(v)
				fmt.Fprint(tw, string(b))
			}
		}
		fmt.Fprintln(tw)
	}
	return nil
}
