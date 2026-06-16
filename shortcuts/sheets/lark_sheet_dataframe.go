// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/ipc"
	"github.com/apache/arrow/go/v17/arrow/memory"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/shortcuts/common"
)

// ─── --dataframe (Arrow IPC / Feather v2 binary input) ────────────────
//
// --dataframe is the binary-typed twin of --sheets. The wire payload is one
// Arrow IPC file (a.k.a. Feather v2 — what `pandas.DataFrame.to_feather()`
// writes), single schema, optionally multi-batch. Type / format are read off
// the Arrow schema (no separate dtypes/formats maps), and per-column number
// format can be set via the field's `number_format` metadata key:
//
//	pa.field("price", pa.float64(), metadata={b"number_format": b"$#,##0.00"})
//
// One DataFrame writes into one sub-sheet at fixed defaults: name `Sheet1`
// (adopted in place by +workbook-create; created when absent by +table-put),
// overwrite from A1 with header on, allow_overwrite=true. The shortcut
// surface is deliberately the one flag — anything that needs a different
// sheet name / anchor / mode / multi-sheet falls back to --sheets, whose
// JSON payload already carries every knob.
//
// Binary IO note: --dataframe bypasses the text-oriented Input resolver
// (`runtime.Str("dataframe")` carries a *path*, not file contents). Reading
// the Arrow bytes through that resolver would TrimSpace the trailing IPC
// magic / corrupt non-UTF8 bytes. Path → FileIO.Open → io.ReadAll keeps the
// stream byte-exact. "-" reads from stdin directly.

// dataframeDefaultSheetName is the sub-sheet name --dataframe writes into.
// Matches valuesSheetName so +workbook-create adopts the brand-new
// workbook's default sheet in place (no stray empty Sheet1 left behind);
// +table-put creates Sheet1 if it doesn't already exist.
const dataframeDefaultSheetName = valuesSheetName

// parseDataframePayload reads the --dataframe path (Arrow IPC file) and
// composes a single-sheet tablePayload at the fixed default placement.
// Network-free: safe from Validate and DryRun. The resulting tableSheetSpec
// rides the same buildSheetMatrix / buildTypedCell path as a --sheets entry,
// so downstream is unaware of where the rows came from.
func parseDataframePayload(rctx *common.RuntimeContext) (*tablePayload, error) {
	raw := strings.TrimSpace(rctx.Str("dataframe"))
	if raw == "" {
		return nil, common.ValidationErrorf("--dataframe is required")
	}
	data, err := readDataframeBytes(rctx, raw)
	if err != nil {
		return nil, err
	}
	spec, err := decodeArrowToSheet(data, dataframeDefaultSheetName)
	if err != nil {
		return nil, common.ValidationErrorf("--dataframe: %v", err)
	}
	payload := &tablePayload{Sheets: []tableSheetSpec{spec}}
	if err := payload.validate(); err != nil {
		return nil, err
	}
	return payload, nil
}

// dataframeStdinCache holds the bytes read from stdin on the first call so a
// later call (Validate → Execute / DryRun) gets the same bytes instead of an
// empty stream — stdin is single-shot, but parseDataframePayload runs
// multiple times per command invocation. Process-wide is fine: lark-cli is
// one-shot (one command per process). Tests reset by setting it back to nil.
var dataframeStdinCache []byte

// readDataframeBytes resolves --dataframe to raw binary. A literal `@` prefix
// is tolerated for symmetry with --sheets (`@/tmp/x.arrow` and `/tmp/x.arrow`
// both work). `-` reads stdin verbatim — cached on first call so Validate /
// Execute / DryRun all see the same bytes. Bytes are returned untouched: no
// TrimSpace, no BOM strip — both would corrupt an Arrow IPC stream.
func readDataframeBytes(rctx *common.RuntimeContext, raw string) ([]byte, error) {
	if raw == "-" {
		if dataframeStdinCache != nil {
			return dataframeStdinCache, nil
		}
		io := rctx.IO()
		if io == nil || io.In == nil {
			return nil, common.ValidationErrorf("--dataframe: stdin is not available")
		}
		data, err := readAllBytes(io.In)
		if err != nil {
			return nil, common.ValidationErrorf("--dataframe: read stdin: %v", err)
		}
		if len(data) == 0 {
			return nil, common.ValidationErrorf("--dataframe: stdin is empty")
		}
		dataframeStdinCache = data
		return data, nil
	}
	path := strings.TrimPrefix(raw, "@")
	data, err := cmdutil.ReadInputFile(rctx.FileIO(), path)
	if err != nil {
		return nil, common.ValidationErrorf("--dataframe: %v", err)
	}
	if len(data) == 0 {
		return nil, common.ValidationErrorf("--dataframe: file %q is empty", path)
	}
	return data, nil
}

// readAllBytes is a thin wrapper so tests can fake the io.Reader without
// importing io. Mirrors io.ReadAll exactly.
func readAllBytes(r io.Reader) ([]byte, error) { return io.ReadAll(r) }

// decodeArrowToSheet reads `data` as an Arrow IPC file (single schema,
// possibly multi-batch) and produces a tableSheetSpec with name + columns +
// rows filled in. Sheet placement (start_cell / mode / header / overwrite) is
// not touched here — parseDataframePayload layers those on from CLI flags.
func decodeArrowToSheet(data []byte, sheetName string) (tableSheetSpec, error) {
	reader, err := ipc.NewFileReader(bytes.NewReader(data))
	if err != nil {
		return tableSheetSpec{}, fmt.Errorf("invalid Arrow IPC file (expected pandas df.to_feather output): %v", err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}
	defer reader.Close()

	schema := reader.Schema()
	if schema == nil || schema.NumFields() == 0 {
		return tableSheetSpec{}, fmt.Errorf("Arrow schema has no fields") //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}

	ncols := schema.NumFields()
	cols := make([]tableColumnSpec, ncols)
	seen := make(map[string]bool, ncols)
	for i := 0; i < ncols; i++ {
		f := schema.Field(i)
		name := f.Name
		if strings.TrimSpace(name) == "" {
			return tableSheetSpec{}, fmt.Errorf("column %d has empty name", i) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		if seen[name] {
			return tableSheetSpec{}, fmt.Errorf("duplicate column name %q", name) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		seen[name] = true
		typ, format, err := arrowFieldToTypeFormat(f)
		if err != nil {
			return tableSheetSpec{}, fmt.Errorf("column %q: %v", name, err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		cols[i] = tableColumnSpec{Name: name, Type: typ, Format: format}
	}

	var rows [][]interface{}
	for b := 0; b < reader.NumRecords(); b++ {
		rec, err := reader.RecordAt(b)
		if err != nil {
			return tableSheetSpec{}, fmt.Errorf("read record batch %d: %v", b, err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		batchRows, err := arrowRecordToRows(rec, cols)
		rec.Release()
		if err != nil {
			return tableSheetSpec{}, err
		}
		rows = append(rows, batchRows...)
	}

	return tableSheetSpec{Name: sheetName, Columns: cols, Rows: rows}, nil
}

// arrowFieldToTypeFormat maps an Arrow field to the internal (type, format)
// pair. The field's `number_format` metadata key — when present — sets the
// Excel number_format string verbatim; otherwise sensible defaults are
// applied per type (`@` text for strings, `yyyy-mm-dd` for dates).
func arrowFieldToTypeFormat(f arrow.Field) (typ, format string, err error) {
	if v, ok := f.Metadata.GetValue("number_format"); ok {
		format = strings.TrimSpace(v)
	}
	switch f.Type.(type) {
	case *arrow.StringType, *arrow.LargeStringType:
		if format == "" {
			format = "@"
		}
		return "string", format, nil
	case *arrow.BooleanType:
		return "bool", format, nil
	case *arrow.Date32Type, *arrow.Date64Type, *arrow.TimestampType:
		if format == "" {
			format = "yyyy-mm-dd"
		}
		return "date", format, nil
	}
	if isArrowNumericType(f.Type) {
		return "number", format, nil
	}
	return "", "", fmt.Errorf("unsupported Arrow type %s (want string/number/date/bool)", f.Type.Name()) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
}

func isArrowNumericType(t arrow.DataType) bool {
	switch t.ID() {
	case arrow.INT8, arrow.INT16, arrow.INT32, arrow.INT64,
		arrow.UINT8, arrow.UINT16, arrow.UINT32, arrow.UINT64,
		arrow.FLOAT16, arrow.FLOAT32, arrow.FLOAT64:
		return true
	}
	return false
}

// arrowRecordToRows transposes one column-batch into row-major
// [][]interface{} matched to `cols`. Cells are stamped with the same value
// shapes buildTypedCell expects from the JSON path: nil for nulls,
// json.Number for numerics (precision-preserving), `yyyy-mm-dd` strings for
// dates/timestamps, bool for booleans, string for strings.
func arrowRecordToRows(rec arrow.Record, cols []tableColumnSpec) ([][]interface{}, error) {
	if int(rec.NumCols()) != len(cols) {
		return nil, fmt.Errorf("record has %d cols, schema declared %d", rec.NumCols(), len(cols)) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}
	nrows := int(rec.NumRows())
	rows := make([][]interface{}, nrows)
	for r := range rows {
		rows[r] = make([]interface{}, len(cols))
	}
	for c := range cols {
		arr := rec.Column(c)
		for r := 0; r < nrows; r++ {
			if arr.IsNull(r) {
				continue
			}
			v, err := arrowCellValue(arr, r)
			if err != nil {
				return nil, fmt.Errorf("row %d column %q: %v", r, cols[c].Name, err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
			}
			rows[r][c] = v
		}
	}
	return rows, nil
}

func arrowCellValue(arr arrow.Array, i int) (interface{}, error) {
	switch a := arr.(type) {
	case *array.String:
		return a.Value(i), nil
	case *array.LargeString:
		return a.Value(i), nil
	case *array.Boolean:
		return a.Value(i), nil
	case *array.Int8:
		return json.Number(strconv.FormatInt(int64(a.Value(i)), 10)), nil
	case *array.Int16:
		return json.Number(strconv.FormatInt(int64(a.Value(i)), 10)), nil
	case *array.Int32:
		return json.Number(strconv.FormatInt(int64(a.Value(i)), 10)), nil
	case *array.Int64:
		return json.Number(strconv.FormatInt(a.Value(i), 10)), nil
	case *array.Uint8:
		return json.Number(strconv.FormatUint(uint64(a.Value(i)), 10)), nil
	case *array.Uint16:
		return json.Number(strconv.FormatUint(uint64(a.Value(i)), 10)), nil
	case *array.Uint32:
		return json.Number(strconv.FormatUint(uint64(a.Value(i)), 10)), nil
	case *array.Uint64:
		return json.Number(strconv.FormatUint(a.Value(i), 10)), nil
	case *array.Float16:
		return json.Number(strconv.FormatFloat(float64(a.Value(i).Float32()), 'f', -1, 32)), nil
	case *array.Float32:
		return json.Number(strconv.FormatFloat(float64(a.Value(i)), 'f', -1, 32)), nil
	case *array.Float64:
		return json.Number(strconv.FormatFloat(a.Value(i), 'f', -1, 64)), nil
	case *array.Date32:
		// Date32: days since 1970-01-01 (epoch). Multiply to seconds, format
		// in UTC so timezone offset can't flip the calendar date.
		t := time.Unix(int64(a.Value(i))*86400, 0).UTC()
		return t.Format("2006-01-02"), nil
	case *array.Date64:
		t := time.UnixMilli(int64(a.Value(i))).UTC()
		return t.Format("2006-01-02"), nil
	case *array.Timestamp:
		ts := int64(a.Value(i))
		unit := a.DataType().(*arrow.TimestampType).Unit
		var t time.Time
		switch unit {
		case arrow.Second:
			t = time.Unix(ts, 0).UTC()
		case arrow.Millisecond:
			t = time.UnixMilli(ts).UTC()
		case arrow.Microsecond:
			t = time.UnixMicro(ts).UTC()
		case arrow.Nanosecond:
			t = time.Unix(0, ts).UTC()
		default:
			return nil, fmt.Errorf("unsupported timestamp unit %v", unit) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		return t.Format("2006-01-02"), nil
	}
	return nil, fmt.Errorf("unsupported Arrow array %T", arr) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
}

// ─── --dataframe-out (Arrow IPC binary output, mirror of --dataframe) ──
//
// +table-get's binary read-back: encode one sheet's typed read-back as an
// Arrow IPC file (Feather v2), so pandas can `pd.read_feather(path)` /
// `pd.read_feather(BytesIO(stdout))` symmetrically with the put side.
// Single-sheet only — Arrow IPC carries one schema per file. The JSON path
// is unchanged; --dataframe-out swaps the encoder for callers that already
// have pandas / pyarrow in their pipeline.

// encodeSheetMapToArrowIPC turns one readSheetAsSpec output into an Arrow IPC
// file blob. Internal column types are recovered from `dtypes` (the wire
// proxy for the typed protocol), and per-column `number_format` rides through
// as Arrow field metadata so the file feeds straight back into
// `+table-put --dataframe`.
func encodeSheetMapToArrowIPC(sheet map[string]interface{}) ([]byte, error) {
	columns, _ := sheet["columns"].([]interface{})
	if len(columns) == 0 {
		return nil, fmt.Errorf("sheet has no columns") //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}
	dtypes, _ := sheet["dtypes"].(map[string]interface{})
	formats, _ := sheet["formats"].(map[string]interface{})
	// `data` arrives as either []interface{} (when the sheet came through a
	// JSON round-trip / unit-test fixture) or [][]interface{} (the shape
	// readSheetAsSpec directly emits in production). Accept both — anything
	// else falls through to a zero-row table.
	var rawData [][]interface{}
	switch d := sheet["data"].(type) {
	case [][]interface{}:
		rawData = d
	case []interface{}:
		rawData = make([][]interface{}, len(d))
		for i, r := range d {
			rawData[i], _ = r.([]interface{})
		}
	}

	ncols := len(columns)
	colNames := make([]string, ncols)
	colTypes := make([]string, ncols)
	fields := make([]arrow.Field, ncols)
	for i, c := range columns {
		name, _ := c.(string)
		if name == "" {
			return nil, fmt.Errorf("column %d has empty name", i) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		colNames[i] = name
		dt, _ := dtypes[name].(string)
		colTypes[i] = dtypeToInternalType(dt)
		var meta arrow.Metadata
		if formats != nil {
			if nf, ok := formats[name].(string); ok && strings.TrimSpace(nf) != "" {
				meta = arrow.NewMetadata([]string{"number_format"}, []string{nf})
			}
		}
		fields[i] = arrow.Field{
			Name:     name,
			Type:     internalTypeToArrowType(colTypes[i]),
			Nullable: true,
			Metadata: meta,
		}
	}
	schema := arrow.NewSchema(fields, nil)

	mem := memory.NewGoAllocator()
	rb := array.NewRecordBuilder(mem, schema)
	defer rb.Release()
	for r, row := range rawData {
		if len(row) != ncols {
			return nil, fmt.Errorf("row %d has %d cells, want %d", r, len(row), ncols) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		for c := 0; c < ncols; c++ {
			if err := appendArrowCell(rb.Field(c), colTypes[c], row[c]); err != nil {
				return nil, fmt.Errorf("row %d column %q: %v", r, colNames[c], err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
			}
		}
	}
	rec := rb.NewRecord()
	defer rec.Release()

	var buf bytesWriterSeeker
	w, err := ipc.NewFileWriter(&buf, ipc.WithSchema(schema), ipc.WithAllocator(mem))
	if err != nil {
		return nil, fmt.Errorf("ipc.NewFileWriter: %v", err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}
	if err := w.Write(rec); err != nil {
		return nil, fmt.Errorf("write record: %v", err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %v", err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}
	return buf.buf, nil
}

// dtypeToInternalType inverts typeToDtype so the Arrow encoder can pick an
// internal column type from the wire-level dtype string. Unknown / object
// falls back to string (lossless: every cell is already typed as such).
func dtypeToInternalType(dtype string) string {
	switch strings.ToLower(strings.TrimSpace(dtype)) {
	case "float64", "float32", "int64", "int32", "int16", "int8",
		"uint64", "uint32", "uint16", "uint8":
		return "number"
	case "bool", "boolean":
		return "bool"
	}
	if strings.HasPrefix(strings.ToLower(dtype), "datetime") {
		return "date"
	}
	return "string"
}

// internalTypeToArrowType is the put-side dtypeToTypeFormat dual: maps the
// internal column type to the Arrow data type the encoder builds a column
// with. Numbers go to float64 because +table-get can't tell int from float
// from a number_format alone — float64 covers both losslessly for the cell
// ranges Lark Sheets accepts.
func internalTypeToArrowType(typ string) arrow.DataType {
	switch typ {
	case "number":
		return arrow.PrimitiveTypes.Float64
	case "date":
		return arrow.FixedWidthTypes.Date32
	case "bool":
		return arrow.FixedWidthTypes.Boolean
	}
	return arrow.BinaryTypes.String
}

// appendArrowCell stamps one cell into its column builder. Cell shape matches
// what cellToTyped emits on the JSON path: json.Number for numbers, ISO
// `yyyy-mm-dd` string for dates, plain string for strings, bool for bools,
// nil for empty. Anything off-shape errors so the caller doesn't silently
// emit nulls for malformed data.
func appendArrowCell(b array.Builder, typ string, v interface{}) error {
	if v == nil {
		b.AppendNull()
		return nil
	}
	switch typ {
	case "string":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("string expects string value, got %T", v) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		b.(*array.StringBuilder).Append(s)
	case "number":
		f, err := arrowNumber(v)
		if err != nil {
			return err
		}
		b.(*array.Float64Builder).Append(f)
	case "date":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("date expects ISO yyyy-mm-dd string, got %T", v) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("date parse %q: %v", s, err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		b.(*array.Date32Builder).Append(arrow.Date32FromTime(t))
	case "bool":
		bb, ok := v.(bool)
		if !ok {
			return fmt.Errorf("bool expects bool, got %T", v) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		b.(*array.BooleanBuilder).Append(bb)
	default:
		return fmt.Errorf("unsupported internal type %q", typ) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}
	return nil
}

// arrowNumber converts the number cell shape readSheetAsSpec emits
// (json.Number) plus the float fallback to float64 for the Arrow builder.
func arrowNumber(v interface{}) (float64, error) {
	switch n := v.(type) {
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, fmt.Errorf("number parse %q: %v", n.String(), err) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
		}
		return f, nil
	case float64:
		return n, nil
	}
	return 0, fmt.Errorf("number expects numeric value, got %T", v) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
}

// bytesWriterSeeker is a 10-line in-memory io.WriteSeeker for
// ipc.NewFileWriter, which seeks back to patch a footer offset. Using a
// buffer (instead of a temp file or os.Stdout, which isn't seekable) keeps
// --dataframe-out's stdout path zero-IO and stays straightforward.
type bytesWriterSeeker struct {
	buf []byte
	pos int64
}

func (w *bytesWriterSeeker) Write(p []byte) (int, error) {
	end := w.pos + int64(len(p))
	if end > int64(len(w.buf)) {
		w.buf = append(w.buf, make([]byte, end-int64(len(w.buf)))...)
	}
	n := copy(w.buf[w.pos:], p)
	w.pos = end
	return n, nil
}

func (w *bytesWriterSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		w.pos = offset
	case io.SeekCurrent:
		w.pos += offset
	case io.SeekEnd:
		w.pos = int64(len(w.buf)) + offset
	default:
		return 0, fmt.Errorf("unknown whence %d", whence) //nolint:forbidigo // intermediate error; the command layer wraps it into a typed --dataframe/--dataframe-out validation error
	}
	return w.pos, nil
}

// writeDataframeOut dispatches the encoded Arrow bytes to wherever --dataframe-out
// points: `-` → process stdout, `@<path>` or plain path → local file. Symmetric
// with readDataframeBytes on the input side: same `@` tolerance, same TrimPrefix
// semantics, and an absolute path will still get rejected by FileIO's SafePath.
func writeDataframeOut(rctx *common.RuntimeContext, raw string, data []byte) error {
	if raw == "-" {
		out := rctx.IO()
		if out == nil || out.Out == nil {
			return common.ValidationErrorf("--dataframe-out: stdout is not available")
		}
		if _, err := out.Out.Write(data); err != nil {
			return errs.NewInternalError(errs.SubtypeFileIO, "--dataframe-out: write stdout").WithCause(err)
		}
		return nil
	}
	path := strings.TrimPrefix(raw, "@")
	fio := rctx.FileIO()
	if fio == nil {
		return common.ValidationErrorf("--dataframe-out: file output is not available in this context")
	}
	// FileIO.Save validates the path via SafeOutputPath (the same sandbox
	// readDataframeBytes hits on the input side) and writes atomically, so we
	// don't need an extra ValidatePath call here.
	if _, err := fio.Save(path, fileio.SaveOptions{ContentLength: int64(len(data))}, bytes.NewReader(data)); err != nil {
		return errs.NewInternalError(errs.SubtypeFileIO, "--dataframe-out: write %q", path).WithCause(err)
	}
	return nil
}
