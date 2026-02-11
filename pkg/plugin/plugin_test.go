// Copyright (c) 2015, 2026, Oracle and/or its affiliates.

//-----------------------------------------------------------------------------
//
// This software is dual-licensed to you under the Universal Permissive License
// (UPL) 1.0 as shown at https://oss.oracle.com/licenses/upl and Apache License
// 2.0 as shown at http://www.apache.org/licenses/LICENSE-2.0. You may choose
// either license.
//
// If you elect to accept the software under the Apache License, Version 2.0,
// the following applies:
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//-----------------------------------------------------------------------------



package plugin

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"
        "fmt"
        "regexp"
        "errors"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// Helper to create a datasource object
func makeTestDS() *OracleDatasource {
	return &OracleDatasource{
		QueryAuth:      "BASIC",
		DeploymentType: "ADB",
		DbUser:         "user",
		DbHostName:     "host",
		DbPortName:     "1521",
		DbServiceName:  "service",
		secureCredData: backend.DataSourceInstanceSettings{
			DecryptedSecureJSONData: map[string]string{
				"dbPassword": "pw",
			},
		},
	}
}

func makeTestRequest() *backend.QueryDataRequest {
	queryJSON := json.RawMessage(`
	{
		"refId": "fetchLabels",
		"queryLang": "promql",
		"exprProm": "up"
	}`)

	return &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{
				RefID: "A",
				JSON:  queryJSON,
				TimeRange: backend.TimeRange{
					From: time.Unix(1700000000, 0),
					To:   time.Unix(1700003600, 0),
				},
			},
			{
				RefID: "B",
				JSON:  queryJSON,
				TimeRange: backend.TimeRange{
					From: time.Unix(1700000000, 0),
					To:   time.Unix(1700003600, 0),
				},
			},
		},
	}
}


func TestNewOracleDatasource(t *testing.T) {
	payload := map[string]string{
		"queryAuth":       "BASIC",
		"deploymentType":  "ADB",
		"dbUser":          "scott",
		"dbConnectString": "dbhost/service",
		"dbHostName":      "dbhost",
		"dbPortName":      "1521",
		"dbServiceName":   "orclpdb1",
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	settings := backend.DataSourceInstanceSettings{
		JSONData:                raw,
		DecryptedSecureJSONData: map[string]string{"dbPassword": "secret"},
	}

	instance, err := NewOracleDatasource(settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ds, ok := instance.(*OracleDatasource)
	if !ok {
		t.Fatalf("expected *OracleDatasource, got %T", instance)
	}

	if ds.QueryAuth != payload["queryAuth"] {
		t.Errorf("QueryAuth: expected %q, got %q", payload["queryAuth"], ds.QueryAuth)
	}
	if ds.DeploymentType != payload["deploymentType"] {
		t.Errorf("DeploymentType: expected %q, got %q", payload["deploymentType"], ds.DeploymentType)
	}
	if ds.DbUser != payload["dbUser"] {
		t.Errorf("DbUser: expected %q, got %q", payload["dbUser"], ds.DbUser)
	}
	if ds.DbConnectString != payload["dbConnectString"] {
		t.Errorf("DbConnectString: expected %q, got %q", payload["dbConnectString"], ds.DbConnectString)
	}
	if ds.DbHostName != payload["dbHostName"] {
		t.Errorf("DbHostName: expected %q, got %q", payload["dbHostName"], ds.DbHostName)
	}
	if ds.DbPortName != payload["dbPortName"] {
		t.Errorf("DbPortName: expected %q, got %q", payload["dbPortName"], ds.DbPortName)
	}
	if ds.DbServiceName != payload["dbServiceName"] {
		t.Errorf("DbServiceName: expected %q, got %q", payload["dbServiceName"], ds.DbServiceName)
	}
	if ds.secureCredData.DecryptedSecureJSONData["dbPassword"] != "secret" {
		t.Errorf("secure password: expected %q, got %q", "secret", ds.secureCredData.DecryptedSecureJSONData["dbPassword"])
	}
}

// helper to avoid strconv in assertions
func itoa(v int64) string {
	return fmt.Sprintf("%d", v)
}

func TestGetPromQLToSQL(t *testing.T) {
	baseFrom := time.Unix(0, 0)
	promql := "up"
	deploymentType := "default"

	tests := []struct {
		name          string
		from          time.Time
		to            time.Time
		stepStr       string
		expectedStep  int64
		expectedFrom  int64
	}{
		{
			name:         "datapoints less than 720 - step unchanged",
			from:         baseFrom,
			to:           baseFrom.Add(100 * time.Second),
			stepStr:      "1",
			expectedStep: 1,
			expectedFrom: 0,
		},
		{
			name:         "datapoints exactly divisible by 720",
			from:         baseFrom,
			to:           baseFrom.Add(7200 * time.Second), // diff = 7200
			stepStr:      "1",
			expectedStep: 10, // 7200 / 720
			expectedFrom: 0,
		},
		{
			name:         "datapoints greater than 720 not divisible",
			from:         baseFrom,
			to:           baseFrom.Add(7201 * time.Second),
			stepStr:      "1",
			expectedStep: 11, // 7201/720 + 1
			expectedFrom: 0,
		},
		{
			name:         "from timestamp aligned to step",
			from:         time.Unix(100, 0),
			to:           time.Unix(820, 0),
			stepStr:      "10",
			expectedStep: 10,
			expectedFrom: 100,
		},
		{
			name:         "from timestamp not aligned to step",
			from:         time.Unix(105, 0),
			to:           time.Unix(825, 0),
			stepStr:      "10",
			expectedStep: 10,
			expectedFrom: 100, // adjusted down
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := getPromQLToSQL(
				tt.from,
				tt.to,
				promql,
				tt.stepStr,
				deploymentType,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate step
			if !strings.Contains(sql, ","+itoa(tt.expectedStep)+")") {
				t.Fatalf("expected step %d in query, got %s",
					tt.expectedStep, sql)
			}

			// Validate from timestamp
			if !strings.Contains(sql, itoa(tt.expectedFrom)) {
				t.Fatalf("expected from timestamp %d in query, got %s",
					tt.expectedFrom, sql)
			}
		})
	}
}

func TestGetConstants(t *testing.T) {
	tests := []struct {
		name           string
		constName      string
		deploymentType string
		expected       string
	}{
		// ---------------- ADB cases ----------------
		{
			name:           "ADB query_range_str",
			constName:      "query_range_str",
			deploymentType: "ADB",
			expected:       "select DBMS_CLOUD_TELEMETRY_QUERY.promql_range('%s',%d,%d,%d) from dual",
		},
		{
			name:           "ADB query_fetchlabels_str",
			constName:      "query_fetchlabels_str",
			deploymentType: "ADB",
			expected:       "select DBMS_CLOUD_TELEMETRY_QUERY.promql_label('__name__',%d,%d) from dual",
		},
		{
			name:           "ADB adhock_key_query_str",
			constName:      "adhock_key_query_str",
			deploymentType: "ADB",
			expected:       "select DBMS_CLOUD_TELEMETRY_QUERY.promql_label(' ',0,0) from dual",
		},
		{
			name:           "ADB adhock_value_query_str",
			constName:      "adhock_value_query_str",
			deploymentType: "ADB",
			expected:       "select DBMS_CLOUD_TELEMETRY_QUERY.promql_label('%s',0,0) from dual",
		},
		{
			name:           "ADB variable_query_str",
			constName:      "variable_query_str",
			deploymentType: "ADB",
			expected:       "select DBMS_CLOUD_TELEMETRY_QUERY.promql_series('%s',%s,%s) from dual",
		},

		// ---------------- Non-ADB cases ----------------
		{
			name:           "Non-ADB query_range_str",
			constName:      "query_range_str",
			deploymentType: "ONPREM",
			expected:       "select DBMS_TELEMETRY_QUERY.promql_range('%s',%d,%d,%d) from dual",
		},
		{
			name:           "Non-ADB query_fetchlabels_str",
			constName:      "query_fetchlabels_str",
			deploymentType: "ONPREM",
			expected:       "select DBMS_TELEMETRY_QUERY.promql_label('__name__',%d,%d) from dual",
		},
		{
			name:           "Non-ADB adhock_key_query_str",
			constName:      "adhock_key_query_str",
			deploymentType: "ONPREM",
			expected:       "select DBMS_TELEMETRY_QUERY.promql_label(' ',0,0) from dual",
		},
		{
			name:           "Non-ADB adhock_value_query_str",
			constName:      "adhock_value_query_str",
			deploymentType: "ONPREM",
			expected:       "select DBMS_TELEMETRY_QUERY.promql_label('%s',0,0) from dual",
		},
		{
			name:           "Non-ADB variable_query_str",
			constName:      "variable_query_str",
			deploymentType: "ONPREM",
			expected:       "select DBMS_TELEMETRY_QUERY.promql_series('%s',%s,%s) from dual",
		},

		// ---------------- Common cases ----------------
		{
			name:           "sysdate_query_str",
			constName:      "sysdate_query_str",
			deploymentType: "ADB",
			expected:       "select sysdate from dual",
		},
		// ---------------- Default / unknown ----------------
		{
			name:           "unknown constName",
			constName:      "unknown_constant",
			deploymentType: "ADB",
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConstants(tt.constName, tt.deploymentType)
			if result != tt.expected {
				t.Errorf("getConstants(%q, %q) = %q, want %q",
					tt.constName, tt.deploymentType, result, tt.expected)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSecs    int64
		wantNSecs   int64
		expectError bool
	}{
		{
			name:        "only seconds",
			input:       "123",
			wantSecs:    123,
			wantNSecs:   0,
			expectError: false,
		},
		{
			name:        "seconds with milliseconds",
			input:       "123.456",
			wantSecs:    123,
			wantNSecs:   456000000,
			expectError: false,
		},
		{
			name:        "seconds with nanoseconds",
			input:       "123.000000789",
			wantSecs:    123,
			wantNSecs:   789,
			expectError: false,
		},
		{
			name:        "more than two parts",
			input:       "1.2.3",
			wantSecs:    -1,
			wantNSecs:   -1,
			expectError: true,
		},
		{
			name:        "invalid seconds only",
			input:       "abc",
			wantSecs:    -1,
			wantNSecs:   -1,
			expectError: true,
		},
		{
			name:        "invalid seconds with fraction",
			input:       "abc.123",
			wantSecs:    -1,
			wantNSecs:   -1,
			expectError: true,
		},
		{
			name:        "invalid fractional seconds",
			input:       "123.xyz",
			wantSecs:    -1,
			wantNSecs:   -1,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			wantSecs:    -1,
			wantNSecs:   -1,
			expectError: true,
		},
		{
			name:        "just dot",
			input:       ".",
			wantSecs:    -1,
			wantNSecs:   -1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secs, nsecs, err := parseTime(tt.input)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if secs != -1 || nsecs != -1 {
					t.Fatalf("expected (-1, -1), got (%d, %d)", secs, nsecs)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if secs != tt.wantSecs || nsecs != tt.wantNSecs {
				t.Fatalf(
					"parseTime(%q) = (%d, %d), want (%d, %d)",
					tt.input, secs, nsecs, tt.wantSecs, tt.wantNSecs,
				)
			}
		})
	}
}

type testStruct struct {
	Name           string
	Age            int
	secureCredData string
}

func TestDumpStruct(t *testing.T) {
	// Struct input
	dumpStruct(testStruct{
		Name:           "Alice",
		Age:            30,
		secureCredData: "secret",
	}, "test")

	// Map input
	dumpStruct(map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}, "test")

	// Map with secureCredData key
	dumpStruct(map[string]interface{}{
		"secureCredData": "secret",
		"visibleKey":     "value",
	}, "test")

	// Nested map
	dumpStruct(map[string]interface{}{
		"outer": map[string]interface{}{
			"inner": "value",
		},
	}, "test")

	// Empty map
	dumpStruct(map[string]interface{}{}, "test")

	// Unsupported type
	dumpStruct(123, "test")

	// Nil input
	dumpStruct(nil, "test")
}

func TestCustomLogger_NoPanic(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"error"},
		{"warning"},
		{"info"},
		{"debug"},
		{"unknown"}, // should silently do nothing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("customLogger panicked for type %s: %v", tt.name, r)
				}
			}()

			customLogger(tt.name, "test message", "value")
		})
	}
}

func TestIsCharColumn(t *testing.T) {
	tests := map[string]bool{
		"VARCHAR2":  true,
		"CHAR":      true,
		"NVARCHAR2": true,
		"NCHAR":     true,
		"NUMBER":    false,
		"DATE":      false,
		"":          false,
	}

	for dt, expected := range tests {
		if got := isCharColumn(dt); got != expected {
			t.Errorf("isCharColumn(%q) = %v, want %v", dt, got, expected)
		}
	}
}

func TestGetSingleColumnDF_Scan(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	// Mock rows with ONE column
	mockRows := sqlmock.NewRows([]string{"metric_name"}).
		AddRow("cpu").
		AddRow("memory").
		AddRow("disk")

	// Expect query (query text doesn't matter much here)
	mock.ExpectQuery("SELECT .*").
		WillReturnRows(mockRows)

	// Execute query to get *sql.Rows
	rows, err := db.Query("SELECT metric_name FROM telemetry_metrics_data")
	if err != nil {
		t.Fatalf("db.Query: %v", err)
	}
	defer rows.Close()

	var processed int
	var after time.Time

	frames, err := getSingleColumnDF(rows, &processed, &after)
	if err != nil {
		t.Fatalf("getSingleColumnDF: %v", err)
	}

	if processed != 3 {
		t.Fatalf("rowsProcessed = %d, want 3", processed)
	}

	if after.IsZero() {
		t.Fatalf("expected timeAfterQuery to be set")
	}

	if len(frames) != 1 {
		t.Fatalf("frames length = %d, want 1", len(frames))
	}

	field := frames[0].Fields[0]

	if field.Name != "metric_name" {
		t.Fatalf("field name = %q, want metric_name", field.Name)
	}

	b, err := json.Marshal(frames[0])
	if err != nil {
		t.Fatalf("json.Marshal(frame): %v", err)
	}

	s := string(b)

	for _, want := range []string{"cpu", "memory", "disk"} {
		if !strings.Contains(s, want) {
			t.Fatalf("frame JSON missing value %q\nJSON: %s", want, s)
		}
	}
}

func TestGetSingleColumnDF_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock init failed: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"TAG"}).
		AddRow(nil) // causes scan error for string

	mock.ExpectQuery("SELECT .*").WillReturnRows(rows)

	sqlRows, _ := db.Query("SELECT TAG FROM dual")
	defer sqlRows.Close()

	var rowsProcessed int
	var timeAfterQuery time.Time

	_, err = getSingleColumnDF(sqlRows, &rowsProcessed, &timeAfterQuery)
	if err == nil {
		t.Fatal("expected scan error, got nil")
	}
}

func TestGetDataFrameFromRows_SQLTabular(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRows([]string{"TAGS", "VALUE", "TIME"}).
		AddRow("{\"host\":\"server1\"}", "0.1", "1766486300").
		AddRow("{\"host\":\"server2\"}", "0.2", "1766486200").
		AddRow("{\"host\":\"server3\"}", "0.3", "1766486100")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sqlRows, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("db.Query: %v", err)
	}
	defer sqlRows.Close()

	var processed int
	var after time.Time

	frames, execTime, err := getDataFrameFromRows(sqlRows, false, false, "SELECT", "", "query", &processed, &after)
	if err != nil {
		t.Fatalf("getDataFrameFromRows: %v", err)
	}

	if execTime != "" {
		t.Fatalf("execTime = %q, want empty string", execTime)
	}

	if processed != 3 {
		t.Fatalf("rowsProcessed = %d, want 3", processed)
	}

	if after.IsZero() {
		t.Fatalf("expected timeAfterQuery to be set")
	}

	if len(frames) != 1 {
		t.Fatalf("frames length = %d, want 1", len(frames))
	}

	if len(frames[0].Fields) != 3 {
		t.Fatalf("field count = %d, want 3", len(frames[0].Fields))
	}

	if frames[0].Fields[1].Config == nil || frames[0].Fields[1].Config.DisplayNameFromDS != "VALUE" {
		t.Fatalf("unexpected DisplayNameFromDS for VALUE field: %+v", frames[0].Fields[1].Config)
	}

        frame := frames[0]
        // Expected values
        expectedTimes := []int64{
             1766486300,
             1766486200,
             1766486100,
        }
        expectedValues := []string{"0.1", "0.2", "0.3"}
        expectedTags := []string{"{\"host\":\"server1\"}","{\"host\":\"server2\"}", "{\"host\":\"server3\"}"}


        // Validate number of rows
        rowCount := frame.Fields[0].Len()

        // Validate each row
        for i := 0; i < rowCount; i++ {
             tags := frame.Fields[0].At(i)
             ts, _ := frame.Fields[2].At(i).(time.Time)
             val := frame.Fields[1].At(i)
             epoch := ts.UTC().Unix()

             // fmt.Println("val ",val)
             // fmt.Println("ts ",ts)
             if epoch != expectedTimes[i] {
                 t.Fatalf("row %d time = %d, want %d",
                 i, epoch, expectedTimes[i])
              }

              if val != expectedValues[i] {
                   t.Fatalf("row %d value = %s, want %s",
                   i, val, expectedValues[i])
              }

              if  tags != expectedTags[i] {
                 t.Fatalf("row %d time = %d, want %d",
                 i, epoch, expectedTimes[i])
              }
         }
        if err := mock.ExpectationsWereMet(); err != nil {
              t.Fatalf("sqlmock expectations: %v", err)
        }
}

func TestGetDataFrameFromRows_SQLTimeseriesWithMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("METRIC_TIME_EPOCH").OfType("NUMBER", ""),
		sqlmock.NewColumn("METRIC_VALUE").OfType("NUMBER", ""),
		sqlmock.NewColumn("METRIC_NAME").OfType("VARCHAR2", ""),
		sqlmock.NewColumn("METRIC_TAGS").OfType("VARCHAR2", ""),
	).
                AddRow("1700000010", "0.5", "cpu_usage", `{"node":"node-a"}`).
		AddRow("1704153600", "1.5", "cpu_usage", `{"node":"node-a"}`)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sqlRows, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("db.Query: %v", err)
	}
	defer sqlRows.Close()

	var processed int
	var after time.Time

	frames, execTime, err := getDataFrameFromRows(sqlRows, false, true, "SELECT", "node", "query", &processed, &after)
	if err != nil {
		t.Fatalf("getDataFrameFromRows: %v", err)
	}

	if execTime != "" {
		t.Fatalf("execTime = %q, want empty string", execTime)
	}

	if processed != 2 {
		t.Fatalf("rowsProcessed = %d, want 3", processed)
	}

	if after.IsZero() {
		t.Fatalf("expected timeAfterQuery to be set")
	}

	if len(frames[0].Fields) != 2 {
		t.Fatalf("frames length = %d, want 2", len(frames[0].Fields))
	}
        frame := frames[0]
        // Expected values
        expectedTimes := []string{
             "2023-11-14T22:13:30Z",
             "2024-01-02T00:00:00Z",
        }
        expectedValues := []float64{0.5,1.5}

        // Validate number of rows
        rowCount := frame.Fields[0].Len()

        // Validate each row
        for i := 0; i < rowCount; i++ {
              ts := frame.Fields[0].At(i).(time.Time)
              val := frame.Fields[1].At(i).(float64)

             if ts.Format(time.RFC3339) != expectedTimes[i] {
                 t.Fatalf("row %d time = %s, want %s",
                 i, ts.Format(time.RFC3339), expectedTimes[i])
              }

              if val != expectedValues[i] {
                   t.Fatalf("row %d value = %f, want %f",
                   i, val, expectedValues[i])
              }
         }
        if err := mock.ExpectationsWereMet(); err != nil {
               t.Fatalf("sqlmock expectations: %v", err)
         }
}
func TestGetDataFrameFromRows_SQLNumericParseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("METRIC_TIME").OfType("TIMESTAMP", ""),
		sqlmock.NewColumn("HOST").OfType("VARCHAR2", ""),
		sqlmock.NewColumn("VALUE").OfType("NUMBER", ""),
	).
		AddRow("2024-01-01T00:00:00Z", "node-a", "bad-number")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sqlRows, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("db.Query: %v", err)
	}
	defer sqlRows.Close()

	var processed int
	var after time.Time

	frames, execTime, err := getDataFrameFromRows(sqlRows, false, true, "SELECT", "", "query", &processed, &after)
	if err == nil {
		t.Fatalf("expected numeric parse error")
	}

	if !strings.Contains(err.Error(), "invalid numeric value") {
		t.Fatalf("unexpected error: %v", err)
	}

	if execTime != "" {
		t.Fatalf("execTime = %q, want empty string", execTime)
	}

	if processed != 0 {
		t.Fatalf("rowsProcessed = %d, want 0", processed)
	}

	if len(frames) != 0 {
		t.Fatalf("frames length = %d, want 0", len(frames))
	}

	//if !after.IsZero() {
	//	t.Fatalf("timeAfterQuery should be zero, got %v", after)
	//}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}

func TestGetDataFrameFromRows_SQLTimeseriesMissingLabelColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("METRIC_TIME").OfType("TIMESTAMP", ""),
		sqlmock.NewColumn("VALUE").OfType("NUMBER", ""),
	).
		AddRow("2024-01-01T00:00:00Z", "1.0")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sqlRows, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("db.Query: %v", err)
	}
	defer sqlRows.Close()

	var processed int
	var after time.Time

	frames, execTime, err := getDataFrameFromRows(sqlRows, false, true, "SELECT", "", "query", &processed, &after)
	if err == nil {
		t.Fatalf("expected error due to missing character columns")
	}

	if !strings.Contains(err.Error(), "Invalid SQL query") {
		t.Fatalf("unexpected error: %v", err)
	}

	if execTime != "" {
		t.Fatalf("execTime = %q, want empty string", execTime)
	}

	if processed != 0 {
		t.Fatalf("rowsProcessed = %d, want 0", processed)
	}

	if len(frames) != 0 {
		t.Fatalf("frames length = %d, want 0", len(frames))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}

func TestGetDataFrameFromRows_SQLCase1InvalidTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("METRIC_TIME").OfType("TIMESTAMP", ""),
		sqlmock.NewColumn("VALUE").OfType("NUMBER", ""),
	).
		AddRow("not-a-time", "1")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sqlRows, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("db.Query: %v", err)
	}
	defer sqlRows.Close()

	var processed int
	var after time.Time

	frames, execTime, err := getDataFrameFromRows(sqlRows, false, false, "SELECT", "", "query", &processed, &after)
	if err == nil {
		t.Fatalf("expected error due to invalid time value")
	}

	if !strings.Contains(err.Error(), "not-a-time") {
		t.Fatalf("unexpected error: %v", err)
	}

	if execTime != "" {
		t.Fatalf("execTime = %q, want empty string", execTime)
	}

	if processed != 0 {
		t.Fatalf("rowsProcessed = %d, want 0", processed)
	}

	if len(frames) != 0 {
		t.Fatalf("frames length = %d, want 0", len(frames))
	}

	//if !after.IsZero() {
	//	t.Fatalf("timeAfterQuery should be zero, got %v", after)
	//}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}

func TestGetDataFrameFromRows_SQLTimeseriesDerivedLabels(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("METRIC_TIME").OfType("TIMESTAMP", ""),
		sqlmock.NewColumn("NODE").OfType("VARCHAR2", ""),
		sqlmock.NewColumn("VALUE").OfType("NUMBER", ""),
	).
		AddRow("2024-01-01T00:00:00Z", "node-a", "1").
		AddRow("1700000100", "node-a", "2.5")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sqlRows, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("db.Query: %v", err)
	}
	defer sqlRows.Close()

	var processed int
	var after time.Time

	frames, execTime, err := getDataFrameFromRows(sqlRows, false, true, "SELECT", "", "query", &processed, &after)
	if err != nil {
		t.Fatalf("getDataFrameFromRows: %v", err)
	}

	if execTime != "" {
		t.Fatalf("execTime = %q, want empty string", execTime)
	}

	if processed != 2 {
		t.Fatalf("rowsProcessed = %d, want 2", processed)
	}

	if after.IsZero() {
		t.Fatalf("expected timeAfterQuery to be set")
	}

	if len(frames) != 1 {
		t.Fatalf("frames length = %d, want 1", len(frames))
	}
        frame := frames[0]
        expectedTimes := []string{"2023-11-14T22:15:00Z", "2024-01-01T00:00:00Z"}
        expectedValues := []float64{2.5,1}

        // Validate number of rows
        rowCount := frame.Fields[0].Len()

        // Validate each row
        for i := 0; i < rowCount; i++ {
              ts := frame.Fields[0].At(i).(time.Time)
              val := frame.Fields[1].At(i).(float64)

             if ts.Format(time.RFC3339) != expectedTimes[i] {
                 t.Fatalf("row %d time = %s, want %s",
                 i, ts.Format(time.RFC3339), expectedTimes[i])
              }

              if val != expectedValues[i] {
                   t.Fatalf("row %d value = %f, want %f",
                   i, val, expectedValues[i])
              }
         }
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}
func TestGetDataFrameFromRows_PromQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	jsonPayload := `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"__name__":"up","instance":"localhost:9100","job":"node"},"values":[[1700000000,"1"],[1700000060,"2"]]}]}}`
	rows := sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("PROM_RESULT").OfType("CLOB", ""),
	).AddRow(jsonPayload)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sqlRows, err := db.Query("SELECT")
	if err != nil {
		t.Fatalf("db.Query: %v", err)
	}
	defer sqlRows.Close()

	originalNow := now
	frozen := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	now = func() time.Time { return frozen }
	t.Cleanup(func() { now = originalNow })

	var processed int
	var after time.Time

	frames, execTime, err := getDataFrameFromRows(sqlRows, true, false, "up", "{{instance}}", "query", &processed, &after)
	if err != nil {
		t.Fatalf("getDataFrameFromRows: %v", err)
	}

	if execTime != "0" {
		t.Fatalf("execTime = %q, want \"0\"", execTime)
	}

	if processed != 2 {
		t.Fatalf("rowsProcessed = %d, want 2", processed)
	}

	if !after.Equal(frozen) {
		t.Fatalf("timeAfterQuery = %v, want %v", after, frozen)
	}

	if len(frames) != 1 {
		t.Fatalf("frames length = %d, want 1", len(frames))
	}

	frame := frames[0]
	if len(frame.Fields) != 2 {
		t.Fatalf("field count = %d, want 2", len(frame.Fields))
	}

	if frame.Fields[1].Config == nil || frame.Fields[1].Config.DisplayNameFromDS != "localhost:9100" {
		t.Fatalf("unexpected legend field config: %+v", frame.Fields[1].Config)
	}

        // Expected values
        expectedTimes := []string{
	      "2023-11-14T22:13:20Z", "2023-11-14T22:14:20Z"}
        expectedValues := []float64{1,2}

        // Validate number of rows
        rowCount := frame.Fields[0].Len()

        // Validate each row
        for i := 0; i < rowCount; i++ {
              ts := frame.Fields[0].At(i).(time.Time)
              val := frame.Fields[1].At(i).(float64)

             if ts.Format(time.RFC3339) != expectedTimes[i] {
                 t.Fatalf("row %d time = %s, want %s",
                 i, ts.Format(time.RFC3339), expectedTimes[i])
              }

              if val != expectedValues[i] {
                   t.Fatalf("row %d value = %f, want %f",
                   i, val, expectedValues[i])
              }
         }
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}

func TestQueryData_BASICAuth(t *testing.T) {
	ctx := context.Background()

	// ---- sqlmock setup ----
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mockRows := sqlmock.NewRows([]string{"metric_name"}).
		AddRow("cpu").
		AddRow("memory")

  mock.ExpectQuery(regexp.QuoteMeta("select DBMS_CLOUD_TELEMETRY_QUERY.promql_label ")).WillReturnRows(mockRows)
	// ---- override dbConnector ----
	orig := dbConnector
	dbConnector = func(_ string) (*sql.DB, error) {
		return db, nil
	}
	defer func() { dbConnector = orig }()

	// ---- datasource ----
	ds := makeTestDS()

	// ---- call ----
	resp, err := ds.QueryData(ctx, makeTestRequest())
	if err != nil {
		t.Fatalf("QueryData returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}

	// ---- verify sqlmock ----
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}


func TestQueryData_NonBASICAuth(t *testing.T) {
	ctx := context.Background()

	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery(".*").WillReturnRows(
		sqlmock.NewRows([]string{"metric"}).AddRow("cpu"),
	)
	mock.ExpectQuery(".*").WillReturnRows(
		sqlmock.NewRows([]string{"metric"}).AddRow("mem"),
	)

	orig := dbConnector
	dbConnector = func(_ string) (*sql.DB, error) { return db, nil }
	defer func() { dbConnector = orig }()

	ds := makeTestDS()
	ds.QueryAuth = "TOKEN"
	ds.DbConnectString = "myconnect"

	resp, err := ds.QueryData(ctx, makeTestRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestQueryData_DBFailure(t *testing.T) {
	origConnector := dbConnector
	defer func() { dbConnector = origConnector }()

	dbConnector = func(string) (*sql.DB, error) {
		return nil, errors.New("db down")
	}

	ds := &OracleDatasource{
		QueryAuth: "BASIC",
		secureCredData: backend.DataSourceInstanceSettings{
			DecryptedSecureJSONData: map[string]string{
				"dbPassword": "pw",
			},
		},
	}

	req := &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{RefID: "A"},
		},
	}

	resp, err := ds.QueryData(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if len(resp.Responses) != 0 {
		t.Fatalf("expected no responses on error")
	}
}

func TestQuery_FetchLabels_Time(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mockRows := sqlmock.NewRows([]string{"metric_name"}).
		AddRow("cpu").
		AddRow("memory")

  expectedSQL := `select DBMS_CLOUD_TELEMETRY_QUERY.promql_label('__name__',1700000000,1700003600) from dual`

  mock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
	         WillReturnRows(mockRows)

	queryJSON := json.RawMessage(`{"refId":"fetchLabels","queryLang":"promql","exprProm":"up","timeFrom":"1700000000000","timeTo":"1700003600000"}`)
       fmt.Println("queryjson ", queryJSON)
	dataQuery := backend.DataQuery{
		RefID: "A",
		JSON:  queryJSON,
		TimeRange: backend.TimeRange{
			From: time.Unix(1700000000, 0),
			To:   time.Unix(1700003600, 0),
		},
	}

	resp := query(dataQuery, db, "ADB")
	if resp.Error != nil {
		t.Fatalf("unexpected response error: %v", resp.Error)
	}
	if len(resp.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(resp.Frames))
	}

	frameJSON, err := json.Marshal(resp.Frames[0])
	if err != nil {
		t.Fatalf("json.Marshal(frame): %v", err)
	}
	frameStr := string(frameJSON)
	for _, want := range []string{"cpu", "memory"} {
		if !strings.Contains(frameStr, want) {
			t.Fatalf("frame JSON missing value %q\nJSON: %s", want, frameStr)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}

func TestQuery_FetchLabels_NoTime(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    rows := sqlmock.NewRows([]string{"label"}).
        AddRow("host")

     mock.ExpectQuery(`select\s+DBMS_TELEMETRY_QUERY\.promql_label`,).
             WillReturnRows(rows)

    qMap := map[string]interface{}{
        "refId":     "fetchLabels",
        "queryLang": "promql",        // REQUIRED
        "exprProm":  "up",             // REQUIRED
    }

    jsonBytes, _ := json.Marshal(qMap)

    q := backend.DataQuery{
        JSON: jsonBytes,
    }

    response := query(q, db, "test")

    if response.Error != nil {
        t.Fatalf("unexpected error")
    }
}

func TestQuery_MetricFindQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"metric_name"}).
		AddRow("region")

	mock.ExpectQuery(`(?i)select\s+DBMS_TELEMETRY_QUERY\.`).
		WillReturnRows(rows)

	qMap := map[string]interface{}{
		"refId":      "metricFindQuery",
		"exprProm":   "cpu_usage",
                "expr":       "cpu_usage&start=1768542262&end=1768542462",
		"queryLang":  "promql",
	}

	jsonBytes, _ := json.Marshal(qMap)

	q := backend.DataQuery{
		RefID: "metricFindQuery",
		JSON:  jsonBytes,
	}

	resp := query(q, db, "test")

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if len(resp.Frames) == 0 {
		t.Fatalf("expected frames in response")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestQuery_getKeys(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    rows := sqlmock.NewRows([]string{"label"}).
        AddRow("host")

     mock.ExpectQuery(`select\s+DBMS_TELEMETRY_QUERY\.promql_label`,).
             WillReturnRows(rows)

    qMap := map[string]interface{}{
	    "refId":"getKeysForAdHocFilter",
	    "exprProm":   "host",
      "expr":       "host",
	    "queryLang":  "promql",
    }
    jsonBytes, _ := json.Marshal(qMap)

    q := backend.DataQuery{
        JSON: jsonBytes,
    }

    resp := query(q, db, "test")

    if resp.Error != nil {
        t.Fatalf("unexpected error")
    }
}

func TestQuery_getValue(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    rows := sqlmock.NewRows([]string{"label"}).
        AddRow("host")

     mock.ExpectQuery(`select\s+DBMS_TELEMETRY_QUERY\.promql_label`,).
             WillReturnRows(rows)

    qMap := map[string]interface{} {
			"refId":      "getValueforKeyAdHocFilter",
			"exprProm":   "host",
			"expr":       "host",
			"queryLang":  "promql",
			"rawQueryText": "cpu",
    }
    jsonBytes, _ := json.Marshal(qMap)

    q := backend.DataQuery{
        JSON: jsonBytes,
    }

    resp := query(q, db, "test")

    if resp.Error != nil {
        t.Fatalf("unexpected error")
    }
}

func TestQuery_PromQL(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
    t.Cleanup(func() {
		_ = db.Close()
	})

    jsonPayload := `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"__name__":"up","instance":"localhost:9100","job":"node"},"values":[[1700000000,"1"],[1700000060,"2"]]}]}}`
    rows := sqlmock.NewRowsWithColumnDefinition(
	      sqlmock.NewColumn("PROM_RESULT").OfType("CLOB", ""),
	).AddRow(jsonPayload)

    mock.ExpectQuery(`select\s+DBMS_TELEMETRY_QUERY\.promql`,).
	  WillReturnRows(rows)

    qMap := map[string]interface{} {
			"refId":      "A",
			"exprProm":   "upinstance",
			"queryLang":  "promql",
    }
    jsonBytes, _ := json.Marshal(qMap)

    q := backend.DataQuery{
        JSON: jsonBytes,
	  TimeRange: backend.TimeRange{
	    From: time.Unix(1700000000, 0),
	    To:   time.Unix(1700003600, 0),
      },
    }

    resp := query(q, db, "test")

    if resp.Error != nil {
        t.Fatalf("unexpected error")
    }
}

func TestCheckHealth_BasicAuth(t *testing.T) {
	orig := dbConnector
	defer func() { dbConnector = orig }()

	tests := []struct {
		name        string
		connector   func(string) (*sql.DB, error)
		expectMsg   string
		expectState backend.HealthStatus
	}{
		{
			name: "basic-success",
			connector: func(string) (*sql.DB, error) {
				db, _, err := sqlmock.New()
				return db, err
			},
			expectMsg:   "Data source is working",
			expectState: backend.HealthStatusOk,
		},
		{
			name: "basic-failure",
			connector: func(string) (*sql.DB, error) {
				return nil, errors.New("boom")
			},
			expectMsg:   "Error Connecting to Database!!! ERROR: boom",
			expectState: backend.HealthStatusError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dbConnector = tc.connector

			ds := &OracleDatasource{
				QueryAuth:      "BASIC",
				DbUser:         "user",
				DbHostName:     "host",
				DbPortName:     "1521",
				DbServiceName:  "service",
				secureCredData: backend.DataSourceInstanceSettings{
					DecryptedSecureJSONData: map[string]string{
						"dbPassword": "pw",
					},
				},
			}

			result, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Status != tc.expectState {
				t.Fatalf("expected status %v, got %v", tc.expectState, result.Status)
			}

			if result.Message != tc.expectMsg {
				t.Fatalf("expected message %q, got %q", tc.expectMsg, result.Message)
			}
		})
	}
}

func TestCheckHealth_NonBasicAuth(t *testing.T) {
	orig := dbConnector
	defer func() { dbConnector = orig }()

	tests := []struct {
		name        string
		connector   func(string) (*sql.DB, error)
		expectMsg   string
		expectState backend.HealthStatus
	}{
		{
			name: "non-basic-success",
			connector: func(string) (*sql.DB, error) {
				db, _, err := sqlmock.New()
				return db, err
			},
			expectMsg:   "Data source is working",
			expectState: backend.HealthStatusOk,
		},
		{
			name: "non-basic-failure",
			connector: func(string) (*sql.DB, error) {
				return nil, errors.New("boom")
			},
			expectMsg:   "Error Connecting to Database!!! ERROR: boom",
			expectState: backend.HealthStatusError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dbConnector = tc.connector

			ds := &OracleDatasource{
				QueryAuth:       "WALLET", // ? forces ELSE branch
				DbUser:          "user",
				DbConnectString: "host:1521/service",
				secureCredData: backend.DataSourceInstanceSettings{
					DecryptedSecureJSONData: map[string]string{
						"dbPassword": "pw",
					},
				},
			}

			result, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Status != tc.expectState {
				t.Fatalf("expected status %v, got %v", tc.expectState, result.Status)
			}

			if result.Message != tc.expectMsg {
				t.Fatalf("expected message %q, got %q", tc.expectMsg, result.Message)
			}
		})
	}
}


func TestSubscribeStream(t *testing.T) {
	ds := &OracleDatasource{}

	tests := []struct {
		name     string
		path     string
		expected backend.SubscribeStreamStatus
	}{
		{
			name:     "allowed path",
			path:     "stream",
			expected: backend.SubscribeStreamStatusOK,
		},
		{
			name:     "denied path",
			path:     "invalid",
			expected: backend.SubscribeStreamStatusPermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &backend.SubscribeStreamRequest{
				Path: tt.path,
			}

			resp, err := ds.SubscribeStream(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Status != tt.expected {
				t.Fatalf("status = %v, want %v", resp.Status, tt.expected)
			}
		})
	}
}

func TestPublishStream(t *testing.T) {
	ds := &OracleDatasource{}

	req := &backend.PublishStreamRequest{
		Path: "stream",
	}

	resp, err := ds.PublishStream(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != backend.PublishStreamStatusPermissionDenied {
		t.Fatalf("status = %v, want PermissionDenied", resp.Status)
	}
}

type testPacketSender struct {
	called int
}

func (t *testPacketSender) Send(_ *backend.StreamPacket) error {
	t.called++
	return nil
}

func TestRunStream_ContextCancel(t *testing.T) {
	ds := &OracleDatasource{}

	ctx, cancel := context.WithCancel(context.Background())

	req := &backend.RunStreamRequest{
		Path: "stream",
	}

	packetSender := &testPacketSender{}
	sender := backend.NewStreamSender(packetSender)

	done := make(chan struct{})

	go func() {
		err := ds.RunStream(ctx, req, sender)
		if err != nil {
			t.Errorf("RunStream returned error: %v", err)
		}
		close(done)
	}()

	// Allow at least one send cycle
	time.Sleep(1100 * time.Millisecond)

	// Cancel stream
	cancel()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("RunStream did not exit after context cancel")
	}

	if packetSender.called == 0 {
		t.Fatal("expected at least one SendFrame call")
	}
}

/*
func TestQueryData_DBError(t *testing.T) {
	orig := dbConnector
	defer func() { dbConnector = orig }()

	dbConnector = func(string) (*sql.DB, error) {
		return nil, fmt.Errorf("mock db error")
	}

	ds := &OracleDatasource{
		QueryAuth: "BASIC",
	}

	_, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

/*
func TestQuery_SQLBranch(t *testing.T) {

	// Prepare query JSON exactly as frontend would send
	queryJSON := map[string]interface{}{
		"queryLang":           "sql",
		"exprSql":             "SELECT tags FROM metrics WHERE $__timeFilter(ts)",
		"legendFormatSql":     "",
                "prefetchCountText":   "200",
		"convertSqlResults":   "true",
		"stepTextSql":         "60",
	}

	jsonBytes, err := json.Marshal(queryJSON)

	if err != nil {
		t.Fatalf("failed to marshal query json: %v", err)
	}

	q := backend.DataQuery{
		JSON: jsonBytes,
		TimeRange: backend.TimeRange{
			From: time.Unix(1700000000, 0),
			To:   time.Unix(1700000600, 0),
		},
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	rows := sqlmock.NewRows([]string{"TAGS", "VALUE", "TIME"}).
		AddRow("{\"host\":\"server1\"}", "0.1", "1766486300").
		AddRow("{\"host\":\"server2\"}", "0.2", "1766486200").
		AddRow("{\"host\":\"server3\"}", "0.3", "1766486100")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	resp := query(q, db, "test")

	// Observable assertion
	if resp.Error != nil {
		t.Fatalf("expected no error, got %v", resp.Error)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
*/


