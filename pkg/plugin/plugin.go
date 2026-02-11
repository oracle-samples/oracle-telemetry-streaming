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



/* Copyright (c) 2013, 2025, Oracle and/or its affiliates. */
/* All rights reserved.*/

/*
   NAME
     plugin.go

   DESCRIPTION
     Backend file of grafana plugin.

   LOCATION
     pkg/plugin/plugin.go

    NOTES
      The file is located in following directory
      grafana/data/plugins/my-plugin/pkg/plugin
*/

package plugin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	godror "github.com/godror/godror"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Make sure OracleDatasource implements required interfaces. This is important
// to do since otherwise we will only get a not implemented error response from
// plugin in runtime. In this example datasource instance implements backend.
// QueryDataHandler, backend.CheckHealthHandler, backend.StreamHandler
// interfaces. Plugin should not implement all these interfaces - only those
// which are required for a particular task.
// For example if plugin does not need streaming functionality then you are
// free to remove methods that implement backend.StreamHandler. Implementing
// instancemgmt.InstanceDisposer is useful to clean up resources used by
// previous datasource instance when a new datasource instance created upon
// datasource settings changed.
var (
	_ backend.QueryDataHandler      = (*OracleDatasource)(nil)
	_ backend.CheckHealthHandler    = (*OracleDatasource)(nil)
	_ backend.StreamHandler         = (*OracleDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*OracleDatasource)(nil)
)

// OracleDatasource is an example datasource which can respond to data queries,
// reports its health and has streaming skills.
type OracleDatasource struct {
	QueryAuth      string
	DeploymentType string
	//for db datasource type
	DbUser          string
	DbConnectString string
	DbHostName      string
	DbPortName      string
	DbServiceName   string
	secureCredData  backend.DataSourceInstanceSettings
}

// NewOracleDatasource creates a new datasource instance.
func NewOracleDatasource(setting backend.DataSourceInstanceSettings) (
	instancemgmt.Instance, error) {
	type JSONData struct {
		QueryAuth      string `json:"queryAuth"`
		DeploymentType string `json:"deploymentType"`
		// For DB datasource type
		DbUser          string `json:"dbUser"`
		DbConnectString string `json:"dbConnectString"`
		DbHostName      string `json:"dbHostName"`
		DbPortName      string `json:"dbPortName"`
		DbServiceName   string `json:"dbServiceName"`
	}
	var jd JSONData
	err := json.Unmarshal(setting.JSONData, &jd)
	if err != nil {
		log.DefaultLogger.Info("Marsheling failed for new datasource", "err", err)
	}
	customLogger("info", "instantiated global object", "")
	customLogger("info", "calling dumpstruct from", "neworacledatasource")

	dumpStruct(jd, "info")
	return &OracleDatasource{
		QueryAuth:      jd.QueryAuth,
		DeploymentType: jd.DeploymentType,
		//for db datasource type
		DbUser:          jd.DbUser,
		DbConnectString: jd.DbConnectString,
		DbHostName:      jd.DbHostName,
		DbPortName:      jd.DbPortName,
		DbServiceName:   jd.DbServiceName,
		secureCredData:  setting,
	}, nil
}

// Function to dump the structure fields and their values
func dumpStruct(s interface{}, dumpctx string) {
	customLogger(dumpctx, "starting the dump", "=================")
	v := reflect.ValueOf(s)
	customLogger(dumpctx, "value curkind", v.Kind())
	customLogger(dumpctx, "value struct", reflect.Struct)
	customLogger(dumpctx, "value map", reflect.Map)
	if v.Kind() == reflect.Struct {
		typeOfS := v.Type()
		customLogger(dumpctx, "dumping Struct", "=================")
		for i := 0; i < v.NumField(); i++ {
			if typeOfS.Field(i).Name != "secureCredData" {
				customLogger(dumpctx, typeOfS.Field(i).Name, v.Field(i).Interface())
			}
		}
		customLogger(dumpctx, "ending the dump", "=================")
		return
	} else if v.Kind() == reflect.Map {
		customLogger(dumpctx, "dumping Map", "=================")
		for _, key := range v.MapKeys() {
			keyStr := key.Interface().(string) // Assuming the key is a string
			if keyStr != "secureCredData" {
				value := v.MapIndex(key)
				// Check if the value is a map itself
				if value.Kind() == reflect.Map {
					customLogger(dumpctx, keyStr, "this is key in map values are====>")
					dumpStruct(value.Interface(), dumpctx) // Recursively print the nested map
				} else {
					valueInterface := value.Interface()
					fmt.Printf("Key: %v, Value: %v\n", keyStr, valueInterface)
				}

				valueInterface := value.Interface()
				customLogger(dumpctx, keyStr, valueInterface)
			}
		}
		customLogger(dumpctx, "ending the dump", "=================")
		return
	}
	customLogger(dumpctx, "error", "Expected a struct or Map")
}

func GetSqlDBWithGoDror(connectionString string) (*sql.DB, error) {
	db, err := sql.Open("godror", connectionString)
	if err != nil {
		logError("error in godror sql.Open: %w", err)
		return db, err
	} else {
		logInfo("successfully opened connection using", "godror")
		queryText := getConstants("sysdate_query_str", "")
		rows, err := db.Query(queryText)
		if err != nil {
			db.Close()
			logError("error in query sql.query: %w", err)
			return db, err
		}
		defer rows.Close()
	}
	return db, err
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a
// new instance created. As soon as datasource settings change detected by SDK
// old datasource instance will be disposed and a new one will be created using
// NewOracleDatasource factory function.
func (jd *OracleDatasource) Dispose() {
	// Clean up datasource instance resources.
	jd = nil
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a
// unique identifier). The QueryDataResponse contains a map of RefID to the
// response for each query, and each response contains Frames ([]*Frame).
func (jd *OracleDatasource) QueryData(
	ctx context.Context,
	req *backend.QueryDataRequest) (
	*backend.QueryDataResponse, error) {
	customLogger("debug", "My QueryData called with request values", req)

	// create response struct
	response := backend.NewQueryDataResponse()

	customLogger("info", "calling dumpstruct from", "QueryData")
	dumpStruct(*jd, "info")
	Auth := jd.QueryAuth
	DeploymentType := jd.DeploymentType
	var dbConn *sql.DB
	var err error
	if Auth == "BASIC" {
		connString := fmt.Sprintf("%s/%s@%s:%s/%s", jd.DbUser, jd.secureCredData.DecryptedSecureJSONData["dbPassword"], jd.DbHostName, jd.DbPortName, jd.DbServiceName)
		dbConn, err = GetSqlDBWithGoDror(connString)
	} else {

		connString := jd.DbUser + "/" + jd.secureCredData.DecryptedSecureJSONData["dbPassword"] + "@" + jd.DbConnectString
		dbConn, err = GetSqlDBWithGoDror(connString)
	}

	if err != nil {
		return response, err
	} else {
		customLogger("info", "My db connection success, now querying", "")
	}
	defer dbConn.Close() 
	// loop over queries and execute them individually.
	for _, curquery := range req.Queries {
		response.Responses[curquery.RefID] = query(curquery, dbConn, DeploymentType)
	}

	return response, nil
}

// custom logging infrastructure.
func logError(logString string, logValue interface{}) {
	log.DefaultLogger.Error(logString, "value", logValue)
}

func logWarning(logString string, logValue interface{}) {
	log.DefaultLogger.Warn(logString, "value", logValue)
}

func logInfo(logString string, logValue interface{}) {
	log.DefaultLogger.Info(logString, "value", logValue)
}

func logDebug(logString string, logValue interface{}) {
	log.DefaultLogger.Debug(logString, "value", logValue)
}

func customLogger(logType string, logString string, logValue interface{}) {
	if logType == "error" {
		logError(logString, logValue)
	} else if logType == "warning" {
		logWarning(logString, logValue)
	} else if logType == "info" {
		logInfo(logString, logValue)
	} else if logType == "debug" {
		logDebug(logString, logValue)
	}
}

// logging query information
func logQueryInfo(queryInfo string, timeInstance string, queryText string) {
	log.DefaultLogger.Info(queryInfo+"::"+timeInstance+"::"+queryText,
		"", "")
	log.DefaultLogger.Info(timeInstance+":Query in first way is:",
		queryText, "!qry ends before'='")
	log.DefaultLogger.Info(timeInstance+":Query in second way is:",
		"query=", queryText)
}

// logging query information with timings
func logQueryStatsInfo(
	queryInfo string,
	timeInstance string,
	queryText string,
	rowsProcessed int,
	timeBeforeQuery time.Time,
	timeAfterQuery time.Time) {

	rtt := timeAfterQuery.Sub(timeBeforeQuery).String()
	processTime := time.Since(timeAfterQuery).String()

	log.DefaultLogger.Info(queryInfo+"::RTT for query::"+queryText,
		"RTT=", rtt)
	log.DefaultLogger.Info("Process Time:", "proctime=", processTime)
	log.DefaultLogger.Info("Total Rows Processed:", "rows_proc=", rowsProcessed)

	logQueryInfo(queryInfo, timeInstance, queryText)
}

// Following functions return true if oracle datatype of column matches
// corresponding datatype which function is expecting.
func isNumberColumn(dataType string) bool {
	if dataType == "NUMBER" || dataType == "DOUBLE" ||
		dataType == "FLOAT" || dataType == "LONG" {
		return true
	}
	return false
}

func isTimeColumn(dataType string, colName string) bool {
	if dataType == "DATE" || dataType == "TIMESTAMP WITH LOCAL TIME ZONE" ||
		dataType == "TIMESTAMP" || dataType == "TIMESTAMP WITH TIME ZONE" ||
		colName == "METRIC_TIME" || colName == "METRIC_TIME_EPOCH" ||
		colName == "TIME" {
		return true
	}
	return false
}

func isCharColumn(dataType string) bool {
	if dataType == "VARCHAR2" || dataType == "CHAR" ||
		dataType == "NVARCHAR2" || dataType == "NCHAR" {
		return true
	}
	return false
}

func getConstants(constName string, deploymentType string) string {
	if deploymentType == "ADB" {
		if constName == "query_range_str" {
			return "select DBMS_CLOUD_TELEMETRY_QUERY.promql_range('%s',%d,%d,%d) from dual"
		}
		if constName == "query_fetchlabels_str" {
			return "select DBMS_CLOUD_TELEMETRY_QUERY.promql_label('__name__',%d,%d) from dual"
		}
		if constName == "adhock_key_query_str" {
			return "select DBMS_CLOUD_TELEMETRY_QUERY.promql_label(' ',0,0) from dual"
		}
		if constName == "adhock_value_query_str" {
			return "select DBMS_CLOUD_TELEMETRY_QUERY.promql_label('%s',0,0) from dual"
		}
		if constName == "variable_query_str" {
			return "select DBMS_CLOUD_TELEMETRY_QUERY.promql_series('%s',%s,%s) from dual"
		}
	} else {
		if constName == "query_range_str" {
			return "select DBMS_TELEMETRY_QUERY.promql_range('%s',%d,%d,%d) from dual"
		}
		if constName == "query_fetchlabels_str" {
			return "select DBMS_TELEMETRY_QUERY.promql_label('__name__',%d,%d) from dual"
		}
		if constName == "adhock_key_query_str" {
			return "select DBMS_TELEMETRY_QUERY.promql_label(' ',0,0) from dual"
		}
		if constName == "adhock_value_query_str" {
			return "select DBMS_TELEMETRY_QUERY.promql_label('%s',0,0) from dual"
		}
		if constName == "variable_query_str" {
			return "select DBMS_TELEMETRY_QUERY.promql_series('%s',%s,%s) from dual"
		}
	}
	if constName == "sysdate_query_str" {
		return "select sysdate from dual"
	}
	if constName == "sqlerror_query_str" {
		return `Invalid SQL query.
                    To plot time series data, the query must return:
                    • exactly one time column
                    • one or more numeric columns
                    • at least one character column (used as series labels)`
	}
	return ""
}

// Following function parses a string of time or timestamp and return time
// in seconds and nanoseconds if possible or error if it cannot.
func parseTime(timeStr string) (int64, int64, error) {
	var err error
	secs := int64(-1)
	nSecs := int64(-1)
	timeSplit := strings.Split(timeStr, ".")

	if len(timeSplit) > 2 {
		//On splitting time with "." we cant have more than two parts thus error.
		customLogger("error", "scan failed to parse time parts case1",
			len(timeSplit))
		customLogger("error", "scanned time value", timeStr)
		err = errors.New("err: splits of time greater than 2")
		return secs, nSecs, err
	} else if len(timeSplit) < 2 {
		//If on splitting, we get 1 part then we only have seconds as part of time
		secs, err = strconv.ParseInt(timeSplit[0], 10, 64)
		if err != nil {
			customLogger("error", "scan failed to parse time parts case2", err)
			customLogger("error", "scanned time value", timeStr)
			secs = -1
			nSecs = -1
			return secs, nSecs, err
		}
		nSecs = 0
		return secs, nSecs, err
	} else {
		//If on splitting, we get 2 part then we have seconds and nanoseconds as
		//part of time.
		secs, err = strconv.ParseInt(timeSplit[0], 10, 64)
		if err != nil {
			customLogger("error", "scan failed to parse time parts case3", err)
			customLogger("error", "scanned time value", timeStr)
			secs = -1
			nSecs = -1
			return secs, nSecs, err
		}
		//Getting nanoseconds from fractional second
		nSecsDur, err2 := time.ParseDuration("0." + timeSplit[1] + "s")
		if err2 != nil {
			customLogger("error", "scan failed to parse time parts case4", err2)
			customLogger("error", "scanned time value", timeStr)
			secs = -1
			nSecs = -1
			return secs, nSecs, err2
		}
		nSecs = nSecsDur.Nanoseconds()
		return secs, nSecs, err
	}
}

// This function converts Promql to proper format so that it can run on our
// database as a sql query
func getPromQLToSQL(from time.Time, to time.Time, promql string,
	stepStr string, deploymentType string) (string, error) {
	var err error
	var timeStr string = strconv.FormatInt(from.Unix(), 10) +
		"_" + strconv.FormatInt(to.Unix(), 10) +
		"_" + strconv.FormatInt((to.Unix()-from.Unix()), 10)

	customLogger("debug", "promql text in getPromQLToSQL", promql)
	customLogger("debug", "Time (From_To_Diff) is", timeStr)

	/* we will manipulate steps here. That is if steps is such that total data
	 * points between from and to is less than 720, we will let it be same but
	 * if number data points is more than 720 between from and two, we will
	 * change value of step such that number of data points gets reduced
	 * to <=720.
	 */
	step, _ := strconv.ParseInt(stepStr, 10, 64)
	dataPoints := (to.Unix() - from.Unix()) / step
	newStep := int64(step)

	customLogger("debug", "original data points", dataPoints)

	if dataPoints > 720.0 {
		if (to.Unix()-from.Unix())%720 == 0 {
			newStep = (to.Unix() - from.Unix()) / 720
			customLogger("debug", "case 1 newstep", newStep)
		} else {
			newStep = (to.Unix()-from.Unix())/720 + 1
			customLogger("debug", "case 2 newstep", newStep)
		}
	}

	//Adjusting from timestamps according to step so that graph appears sliding
	fromTs := from.Unix()
	toTs := to.Unix()
	if (fromTs % step) != 0 {
		remainder := fromTs % step
		fromTs = fromTs - remainder
	}
	queryText := fmt.Sprintf(getConstants("query_range_str", deploymentType), promql, fromTs,
		toTs, newStep)
	logQueryInfo("Query query_range", "Before", queryText)
	customLogger("debug", "returned value from getPromQLToSQL", queryText)
	return queryText, err
}

// This function is called to convert the rows to dataframe for given promql
// which has only one column in it.
func getSingleColumnDF(rows *sql.Rows, rowsProcessed *int, timeAfterQuery *time.Time) (data.Frames, error) {
	customLogger("info", "Rows inside getSingleColumnDF Function", rows)
	// create a frame with only one field for tags
	frames := data.Frames{}
	frame := data.NewFrame("response")
	cols, err := rows.Columns()
	if err != nil {
		customLogger("error", "Failed to get columns error occurred", err)
		return frames, err
	}
	*timeAfterQuery = time.Now()

	// here there is only one coulmn by default is we are projecting only one
	frame.Fields = append(frame.Fields,
		data.NewField(cols[0], nil, []string{}),
	)

	// make result interface
	rawResult := make([]string, len(cols))

	dest := make([]interface{}, len(cols))
	for i := range rawResult {
		// Put pointers to each string in the interface slice
		dest[i] = &rawResult[i]
	}
	// for each row append the value in frame for tags
	rowsTotal := 0
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		err = rows.Scan(dest...)
		if err != nil {
			customLogger("error", "scannning row error", err)
			return frames, err
		}

		for i, raw := range rawResult {
			vals[i] = string(raw)
		}
		frame.AppendRow(vals...)
		rowsTotal = rowsTotal + 1
	}
	*rowsProcessed = rowsTotal
	// Append the result in a single frame and return it.
	frames = append(frames, frame)
	return frames, err
}

// This function converts the rows returned from sql query ti the dataframe
// format so that it can be returned to Grafana in required format
func getDataFrameFromRows(rows *sql.Rows, promqlflg bool,
	convertSqlResults bool, qryInputVal string,
	legendTextVal string, queryTextConverted string,
	rowsProcessed *int, timeAfterQuery *time.Time) (
	data.Frames, string, error) {
	//There can be 3 cases,
	//1st Case: the user gives sql query (promqlflg is false) and doesnot wants
	//          tabular results (convertSqlResults is false)
	//2nd Case: the user gives sql query (promqlflg is false) and wants tabular
	//          results (convertSqlResults is true)
	//3rd Case: the user gives promql query. Here the result is already in
	//          required format of grafana.
	execTime := ""
	if !promqlflg && !convertSqlResults {
		//This is the case 1 that we have seen above
		frames := data.Frames{}
		customLogger("debug", "Inside getDataFrameFromRows Func 1 promqlflg:",
			promqlflg)
		customLogger("debug", "convertSqlResults value", convertSqlResults)
		//create a new Dataframe
		frame := data.NewFrame(queryTextConverted)
		//get columns for current sql rows.
		cols, err := rows.Columns()
		if err != nil {
			customLogger("error", "Failed to get columns error occurred ", err)
			return frames, execTime, err
		}

		//get and count type of columns
		types, err := rows.ColumnTypes()
		if err != nil {
			customLogger("error", "Failed to get column types error occurred ",
				err)
			return frames, execTime, err
		}

		dName := "dataframe"
		if legendTextVal != "" {
			dName = legendTextVal
		}
		*timeAfterQuery = time.Now()
		//Now add fields in dataframe that we created for each column we get
		//in sql rows Notice here that if column type is time , we create
		//field of type time.Time{}, if column type is number, we create
		//field of type float64{} , else we create field of string type
		for i, coltype := range types {
			if isTimeColumn(coltype.DatabaseTypeName(), cols[i]) {
				frame.Fields = append(frame.Fields,
					data.NewField(cols[i],
						nil, []time.Time{}),
				)
			} else {
				if legendTextVal != "" {
					dName = legendTextVal
				} else {
					dName = coltype.Name()
				}
				if isNumberColumn(coltype.DatabaseTypeName()) {
					frame.Fields = append(frame.Fields,
						data.NewField(cols[i],
							nil, []float64{}).SetConfig(
							&data.FieldConfig{DisplayNameFromDS: dName}),
					)
				} else {
					frame.Fields = append(frame.Fields,
						data.NewField(cols[i],
							nil, []string{}).SetConfig(
							&data.FieldConfig{DisplayNameFromDS: dName}),
					)
				}
			}
		}

		customLogger("debug", "fields value", frame.Fields)

		//Rawresult array will be used to fetch each row in for loop
		rawResult := make([]sql.NullString, len(cols))

		//dest array will store pointers to rawResult
		dest := make([]interface{}, len(cols))
		// A temporary interface{} slice
		for i := range rawResult {
			dest[i] = &rawResult[i]
			// Put pointers to each string in the interface slice
		}
		//for each row we iterate, we fetch the tuple value from rows and
		//store in dataframe.
		rowsTotal := 0
		for rows.Next() {
			//this vals array is what will be inserted in dataframe
			vals := make([]interface{}, len(cols))
			// A temporary interface{} slice
			//scan current row in dest array
			//we can then access it with help of rawResult array
			err = rows.Scan(dest...)
			if err != nil {
				customLogger("error", "alert scan row error", err)
				return frames, execTime, err
			}

			//for every element in current row we iterate
			for i, raw := range rawResult {
				//get the current colName and depending on column parse the
				//value of current element as time , float64, or string and
				//store it in vals array
				if isTimeColumn(types[i].DatabaseTypeName(), cols[i]) {
					//customLogger("debug", "raw time value", raw)
					ts, err := time.Parse(time.RFC3339, raw.String)
					if err != nil {
						convraw := raw.String
						if !raw.Valid {
							convraw = "0"
						}
						parsedSecs, parsedNSecs, errTime := parseTime(convraw)
						if errTime == nil && parsedSecs != -1 &&
							parsedNSecs != -1 {
							vals[i] = time.Unix(parsedSecs, parsedNSecs)
						} else {
							customLogger("error", "scan failed to parse timevalue:", errTime)
							return frames, execTime, errTime
						}
					} else {
						vals[i] = ts
					}

				} else if isNumberColumn(types[i].DatabaseTypeName()) {
					convraw := raw.String
					if !raw.Valid {
						convraw = "0"
					}
					vals[i], err = strconv.ParseFloat(convraw, 64)
					if err != nil {
						customLogger("error", "scan failed to parse value:",
							err)
						return frames, execTime, err
					}
				} else {
					vals[i] = string(raw.String)
				}
			}
			//finnaly append the vals array (current row values) in
			//the dataframe
			frame.AppendRow(vals...)
			rowsTotal = rowsTotal + 1
		}
		/* customLogger("debug", "frame returned", frame) */
		*rowsProcessed = rowsTotal
		customLogger("debug", "total rows processed", rowsTotal)
		//append the frame in another dataframe and return the result.
		frames = append(frames, frame)
		return frames, execTime, err
	} else if !promqlflg && convertSqlResults {
		//this is case 2 that we described above
		customLogger("debug", "Inside getDataFrameFromRows Func 2, promqlflg:",
			promqlflg)
		customLogger("debug", "case 2, convertSqlResults", convertSqlResults)
		frames := data.Frames{}
		cols, err := rows.Columns()

		if err != nil {
			customLogger("error", "Failed to get columns case2 error", err)
			return frames, execTime, err
		}
		*timeAfterQuery = time.Now()
		// find if following 4 columns exist in projection
		var flgTimeFound bool = false
		var flgValueFound bool = false
		var flgMetricNameFound bool = false
		var flgTagsFound bool = false
		for _, colName := range cols {
			if colName == "METRIC_TIME_EPOCH" {
				flgTimeFound = true
			} else if colName == "METRIC_VALUE" {
				flgValueFound = true
			} else if colName == "METRIC_NAME" {
				flgMetricNameFound = true
			} else if colName == "METRIC_TAGS" {
				flgTagsFound = true
			}
		}
		// only if all 4 required fields presnet then convert the results of sql
		// in timeseries format else return result in tabular format.
		if flgTagsFound && flgValueFound && flgTimeFound &&
			flgMetricNameFound {
			// all 4 required fields found return in timeseries format
			framesFinal := data.Frames{}
			type MetricTagKey struct {
				MetricStr string
				TagsStr   string // stable JSON encoding
			}

			type TimeValuePair struct {
				TimeStr string
				Value   float64
			}

			customLogger("debug",
				"Inside getDataFrameFromRows Func case 2-1, promqlflg", promqlflg)
			rawResult := make([]sql.NullString, len(cols))
			dest := make([]interface{}, len(cols))
			for i := range rawResult {
				dest[i] = &rawResult[i]
				// Put pointers to each string in the interface slice
			}
			//create a map with key as combination of metric and tags
			// and store list of time value pairs in it.
			rowsTotal := 0
			timeSlice := ""
			valSlice := ""
			metricNameSlice := ""
			metricTagsSlice := ""
			// Map: key -> list of time/value pairs
			timeseriesMap := make(map[MetricTagKey][]TimeValuePair)
			// Track total rows read

			for rows.Next() {
				err = rows.Scan(dest...)
				if err != nil {
					customLogger("error", "scan row error case 2-3", err)
					return frames, execTime, err
				}
				metricNameSlice, metricTagsSlice, timeSlice, valSlice = "", "", "", ""

				for i, raw := range rawResult {
					colName := cols[i]
					if colName == "METRIC_TIME_EPOCH" {
						curStr := raw.String
						if !raw.Valid {
							curStr = "0"
						}
						timeSlice = curStr
					} else if colName == "METRIC_VALUE" {
						curStr := raw.String
						if !raw.Valid {
							curStr = "0"
						}
						valSlice = curStr
					} else if colName == "METRIC_NAME" {
						curStr := raw.String
						if !raw.Valid {
							curStr = ""
						}
						metricNameSlice = curStr
					} else if colName == "METRIC_TAGS" {
						curStr := raw.String
						if !raw.Valid {
							curStr = ""
						}
						metricTagsSlice = curStr
					}
				}
				valNum, err := strconv.ParseFloat(valSlice, 64)
				if err != nil {
					customLogger("error", "Failed to parse value", err)
					return nil, execTime, err
				}

				key := MetricTagKey{MetricStr: metricNameSlice, TagsStr: metricTagsSlice}
				timeseriesMap[key] = append(timeseriesMap[key], TimeValuePair{
					TimeStr: timeSlice,
					Value:   valNum,
				})
				rowsTotal++
			}

			for key, pairs := range timeseriesMap {
				sort.Slice(pairs, func(i, j int) bool {
					t1, err1 := time.Parse(time.RFC3339, pairs[i].TimeStr)
					if err1 != nil {
						sec, nsec, _ := parseTime(pairs[i].TimeStr)
						t1 = time.Unix(sec, nsec)
					}
					t2, err2 := time.Parse(time.RFC3339, pairs[j].TimeStr)
					if err2 != nil {
						sec, nsec, _ := parseTime(pairs[j].TimeStr)
						t2 = time.Unix(sec, nsec)
					}
					return t1.Before(t2)
				})
				timeseriesMap[key] = pairs

				metricName := key.MetricStr
				metricTagsJSON := key.TagsStr

				var tagsMap map[string]string
				var errjson error
				if metricTagsJSON != "" {
					errjson = json.Unmarshal([]byte(metricTagsJSON), &tagsMap)
					if errjson != nil {
						// keep tagsMap nil and continue; we won't fail because tags may be optional
						customLogger("debug", "failed to unmarshal tags json for key", errjson)
						tagsMap = nil
					}
				}
				// Create new frame for this group
				curFrame := data.NewFrame(queryTextConverted)
				curFrame.Fields = append(curFrame.Fields,
					data.NewField("METRIC_TIME", nil, []time.Time{}),
				)

				dName := metricName + metricTagsJSON

				if legendTextVal == "" {
					//customLogger("debug", "legends empty, tags value",
					//                        tagsMap)
					if errjson != nil {
						//customLogger("debug", "Func 2-1, errjson",
						// errjson)
						curFrame.Fields = append(curFrame.Fields,
							data.NewField(dName,
								tagsMap, []float64{}).SetConfig(
								&data.FieldConfig{DisplayNameFromDS: dName}),
						)
					} else {
						curFrame.Fields = append(curFrame.Fields,
							data.NewField(dName,
								nil, []float64{}).SetConfig(
								&data.FieldConfig{DisplayNameFromDS: dName}),
						)
					}

				} else {
					// manipulate the legendVal,if legendVal doesnot contain
					//{$var1,$var2} then do nothing else if legendVal is
					//something like lv{$node,$cpu}, we will replace $node
					//by current value of node and $cpu by current value
					//of cpu
					legendVal := strings.TrimSpace(legendTextVal)
					finalLegend := ""
					if strings.Contains(legendVal, "{{") &&
						strings.Contains(legendVal, "}}") {
						startSep := "{{"
						endSep := "}}"
						tmp := strings.Split(legendVal, startSep)

						for i, ind := range tmp {
							if strings.Contains(ind, endSep) {
								finalLegend = finalLegend +
									strings.Split(tmp[i-1], endSep)[len(strings.Split(tmp[i-1], endSep))-1]
								finalLegend = finalLegend +
									tagsMap[strings.Split(ind, endSep)[0]]
							}
						}
						if !strings.HasSuffix(legendVal, endSep) {
							finalLegend = finalLegend +
								strings.Split(legendVal, endSep)[len(strings.Split(legendVal, endSep))-1]
						}
					} else {
						finalLegend = legendVal
					}

					if errjson != nil {
						//customLogger("debug", "Func 2-3, errjson",
						//errjson)
						curFrame.Fields = append(curFrame.Fields,
							data.NewField(finalLegend,
								tagsMap, []float64{}).SetConfig(
								&data.FieldConfig{DisplayNameFromDS: finalLegend}),
						)
					} else {
						curFrame.Fields = append(curFrame.Fields,
							data.NewField(finalLegend,
								nil, []float64{}).SetConfig(
								&data.FieldConfig{DisplayNameFromDS: finalLegend}),
						)
					}
				}

				for _, pair := range pairs {
					ts, err := time.Parse(time.RFC3339, pair.TimeStr)
					if err != nil {
						sec, nsec, err2 := parseTime(pair.TimeStr)
						if err2 != nil || sec == -1 {
							customLogger("error", "Failed to parse time", err2)
							continue
						}
						ts = time.Unix(sec, nsec)
					}
					curFrame.AppendRow(ts, pair.Value)
				}

				framesFinal = append(framesFinal, curFrame)
			}
			*rowsProcessed = rowsTotal
			customLogger("debug", "rowsTotal:", rowsTotal)
			//return the final frame consisting of all the frames we created
			return framesFinal, execTime, err
		} else {
			// this is the case where user has asked results in timeseries
			// format but has not projected the required 4 columns. In this
			// case we try to find if we can form timeseries else return error
			customLogger("debug",
				"Inside getDataFrameFromRows Func case 2-2 promqlflg", promqlflg)
			numberCols := 0
			timeCols := 0
			charCols := 0
			otherCols := 0
			types, err := rows.ColumnTypes()
			if err != nil {
				customLogger("error", "Failed to get column types error occurred ",
					err)
				return frames, execTime, err
			}
			for i, coltype := range types {
				customLogger("info", "cur col type", coltype.DatabaseTypeName())
				customLogger("info", "cur col name", cols[i])
				if isTimeColumn(coltype.DatabaseTypeName(), cols[i]) {
					timeCols = timeCols + 1
				} else if isNumberColumn(coltype.DatabaseTypeName()) {
					numberCols = numberCols + 1
				} else if isCharColumn(coltype.DatabaseTypeName()) {
					charCols = charCols + 1
				} else {
					otherCols = otherCols + 1
				}
			}
			customLogger("info", "numberCols found count", numberCols)
			customLogger("info", "timeCols found count", timeCols)
			customLogger("info", "charCols found count", charCols)
			customLogger("info", "otherCols found count", otherCols)

			framesFinal := data.Frames{}
			rawResult := make([]sql.NullString, len(cols))
			dest := make([]interface{}, len(cols))

			type TimeValuePair struct {
				TimeStr string
				Value   float64
			}

			//create a map with key as combination of all char columns
			// and store list of time value pairs in it.
			timeseriesMap := make(map[string][]TimeValuePair)

			// In this case, we will club the character columns to form unique
			// timeseries out of them
			if timeCols == 1 && numberCols == 1 && charCols >= 1 {
				customLogger("info",
					"Inside getDataFrameFromRows Function case1 charCols", charCols)
				*timeAfterQuery = time.Now()
				// A temporary interface{} slice
				for i := range rawResult {
					dest[i] = &rawResult[i]
					// Put pointers to each string in the interface slice
				}

				//itearate for each row (tuple) in sql results
				for rows.Next() {
					err = rows.Scan(dest...)
					if err != nil {
						customLogger("error", "scan row error part2-3", err)
						return frames, execTime, err
					}
					//declare key string and val string
					key := "" // concatenated char columns
					var tvp TimeValuePair
					//for each element in row create strings
					//first string will be concatenation of all char columns in row
					//and will act as key in map.
					//second string will be time and value pair which will be
					//inserted in list of pairs in map for the corresponding key.
					for i, raw := range rawResult {
						dbType := types[i].DatabaseTypeName()
						colName := cols[i]
						if isTimeColumn(dbType, colName) {
							if raw.Valid {
								tvp.TimeStr = raw.String
							} else {
								tvp.TimeStr = "0"
							}
							continue
						} else if isNumberColumn(dbType) {
							if raw.Valid {
								v, err := strconv.ParseFloat(raw.String, 64)
								if err != nil {
									return nil, execTime, fmt.Errorf("invalid numeric value '%s' in column %s: %w", raw.String, colName, err)
								}
								tvp.Value = v
							} else {
								tvp.Value = 0
							}
							continue
						} else if isCharColumn(dbType) {
							if raw.Valid {
								key += raw.String
							}
							continue
						}
					}
					//customLogger("info", "map1key", keyColsConcatenated);
					//customLogger("info", "map1value", valueTimeVal);
					//insert key val pair in map tag ts
					timeseriesMap[key] = append(timeseriesMap[key], tvp)

				}
				//customLogger("info", "done map", "done map");
				//print the map
				//customLogger("info", "map1value", len(mapCharColmnsTimeseries));
				//customLogger("info", "done map2", "done map2");
				//now we have map of key as char cols concatenated and list of
				//time value pairs as value we just need to convert it in dataframe
				//format as follows for each key in map, extract the key and value
				//and insert them in dataframe

			} else if timeCols == 1 && numberCols >= 1 && charCols >= 1 {
				customLogger("info",
					"Inside getDataFrameFromRows Function case2 numberCols", numberCols)
				*timeAfterQuery = time.Now()
				// A temporary interface{} slice
				for i := range rawResult {
					dest[i] = &rawResult[i]
					// Put pointers to each string in the interface slice
				}

				//itearate for each row (tuple) in sql results
				for rows.Next() {
					err = rows.Scan(dest...)
					if err != nil {
						customLogger("error", "scan row error part2-3", err)
						return frames, execTime, err
					}
					//declare key string and val string
					key := ""            // concatenated char columns
					keywithcolname := "" // concatenated char columns
					strtime := ""
					//for each element in row create strings
					//first string will be concatenation of all char columns in row
					//and will act as key in map.
					//second string will be time and value pair which will be
					//inserted in list of pairs in map for the corresponding key.
					for i, raw := range rawResult {
						dbType := types[i].DatabaseTypeName()
						colName := cols[i]
						if isTimeColumn(dbType, colName) {
							if raw.Valid {
								strtime = raw.String
							} else {
								strtime = "0"
							}
							continue
						} else if isCharColumn(dbType) {
							if raw.Valid {
								key += raw.String
							}
							continue
						}
					}

					for i, raw := range rawResult {
						dbType := types[i].DatabaseTypeName()
						colName := cols[i]
						if isNumberColumn(dbType) && !isTimeColumn(dbType, colName) {
							keywithcolname = key + colName
							var tvp TimeValuePair
							tvp.TimeStr = strtime
							if raw.Valid {
								v, err := strconv.ParseFloat(raw.String, 64)
								if err != nil {
									return nil, execTime, fmt.Errorf("invalid numeric value '%s' in column %s: %w", raw.String, colName, err)
								}
								tvp.Value = v
								timeseriesMap[keywithcolname] = append(timeseriesMap[keywithcolname], tvp)
							} else {
								tvp.Value = 0
								timeseriesMap[keywithcolname] = append(timeseriesMap[keywithcolname], tvp)

							}
							continue
						}
					}

				}
				//customLogger("info", "done map", "done map");
				//print the map
				//customLogger("info", "map1value", len(mapCharColmnsTimeseries));
				//customLogger("info", "done map2", "done map2");
				//now we have map of key as char cols concatenated and list of
				//time value pairs as value we just need to convert it in dataframe
				//format as follows for each key in map, extract the key and value
				//and insert them in dataframe

			} else {
				//else we return the error
				frames := data.Frames{}
				return frames, execTime, errors.New(getConstants("sqlerror_query_str", ""))
			}
			rowsTotal := 0
			for key, pairs := range timeseriesMap {
				//customLogger("info", "map key", key);
				sort.Slice(pairs, func(i, j int) bool {
					t1, err1 := time.Parse(time.RFC3339, pairs[i].TimeStr)
					if err1 != nil {
						sec, nsec, _ := parseTime(pairs[i].TimeStr)
						t1 = time.Unix(sec, nsec)
					}

					t2, err2 := time.Parse(time.RFC3339, pairs[j].TimeStr)
					if err2 != nil {
						sec, nsec, _ := parseTime(pairs[j].TimeStr)
						t2 = time.Unix(sec, nsec)
					}

					return t1.Before(t2)
				})
				timeseriesMap[key] = pairs

				curFrame := data.NewFrame("response")
				curFrame.Fields = append(curFrame.Fields,
					data.NewField("METRIC_TIME", nil, []time.Time{}),
				)

				dName := key

				if legendTextVal == "" {
					//if no legend present, simply create dataframe with name of
					//key(concat of all char columns)
					curFrame.Fields = append(curFrame.Fields,
						data.NewField(dName,
							nil, []float64{}).SetConfig(
							&data.FieldConfig{DisplayNameFromDS: dName}),
					)
					//customLogger("info", "legends empty, display name:", dName)
				} else {
					//replace name in dataframe as whatever is given in legend
					//customLogger("info", "legend not empty, legend:", legendTextVal)
					curFrame.Fields = append(curFrame.Fields,
						data.NewField(legendTextVal,
							nil, []float64{}).SetConfig(
							&data.FieldConfig{DisplayNameFromDS: legendTextVal}),
					)
				}

				//finally create time and value fields in dataframe with value
				//having tags as the map we created this is how the grafana
				//expects the tags for a dataframe.

				//for each value(time, value) pair in the map, insert a row in
				//dataframe after parsing it in required format
				for _, tv := range pairs {
					raw := tv.TimeStr
					ts, err := time.Parse(time.RFC3339, raw)
					if err != nil {
						parsedSecs, parsedNSecs, errTime := parseTime(raw)
						if errTime == nil && parsedSecs != -1 &&
							parsedNSecs != -1 {
							ts = time.Unix(parsedSecs, parsedNSecs)
						} else {
							customLogger("error", "Failed to parse time", errTime)
							customLogger("error", "Failed to parse time's value", raw)
							return frames, execTime, errTime
						}
					}
					rowsTotal = rowsTotal + 1
					curFrame.AppendRow(ts, tv.Value)
				}
				//append the frame for current metric_tag pair in resultant
				//frame
				framesFinal = append(framesFinal, curFrame)
			}
			*rowsProcessed = rowsTotal
			//return the final frame consisting of all the frames we created
			return framesFinal, execTime, err
		}
	}

	// this is case 3 that we saw above, here promqlflg = true and
	// we need to return dataframe
	customLogger("debug", "Inside getDataFrameFromRows PromPart 3, promqlflg",
		promqlflg)
	frames := data.Frames{}

	//Rawresult array will be used to fetch each row in for loop
	var rawResult string = ""
	var dest *string

	rows.Next()
	dest = &rawResult // Put pointers to each string in the interface slice

	var vals string = ""
	err := rows.Scan(dest)
	if err != nil {
		customLogger("error", "Error in scan", err)
		return frames, execTime, err
	}
	customLogger("debug",
		"Inside getDataFrameFromRows PromPart 3, scan success", err)
	*timeAfterQuery = time.Now()

	type RespStruct struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Values [][]interface{}   `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	vals = string(rawResult) // string json result dump

	var jsn RespStruct
	//json.Unmarshal([]byte(tempVals4), &jsn2)
	errjson := json.Unmarshal([]byte(vals), &jsn)
	if errjson != nil {
		customLogger("error", "Error in Json Unmarshal", errjson)
		return frames, execTime, errjson
	}
	// get the json string to object in "jsn"

	customLogger("info", "vals", vals)
	customLogger("info", "valsjson", jsn)

	execTime = "0"
	rowsTotal := 0
	for _, item := range jsn.Data.Result {
		customLogger("info", "iteration start", "")
		//create a dataframe for current tags
		frame := data.NewFrame("response")
		//prepare tags to add in current timeseries dataframe
		tags := make(map[string]string, len(item.Metric))
		for key, element := range item.Metric {
			tags[key] = element
		}
		// add time and value fields for current dataframe along with tags
		// created
		frame.Fields = append(frame.Fields,
			data.NewField("METRIC_TIME", nil, []time.Time{}),
		)
		tagsEmpty := make(map[string]string)
		if legendTextVal == "" {
			customLogger("info", "legend empty1, tags value", tags)
			customLogger("info", "legend empty2, qryInputVal", qryInputVal)
			customLogger("info", "legend empty3, qryInputVal", tags["__name__"])
			frame.Fields = append(frame.Fields,
				data.NewField(tags["__name__"], tags, []float64{}),
			)
		} else {
			// manipulate the legendVal,
			//if legendVal doesnot contain {{objectname}} then do nothing
			//else if legendVal is something like {{objectname}} ,
			//we will replace objectname by current value of objectname
			legendVal := strings.TrimSpace(legendTextVal)
			finalLegend := ""
			if strings.Contains(legendVal, "{{") &&
				strings.Contains(legendVal, "}}") {
				startSep := "{{"
				endSep := "}}"
				tmp := strings.Split(legendVal, startSep)

				for i, ind := range tmp {
					if strings.Contains(ind, endSep) {
						finalLegend = finalLegend +
							strings.Split(
								tmp[i-1],
								endSep)[len(strings.Split(tmp[i-1], endSep))-1]
						finalLegend = finalLegend +
							tags[strings.Split(ind, endSep)[0]]
					}
				}
				if !strings.HasSuffix(legendVal, endSep) {
					finalLegend = finalLegend +
						strings.Split(
							legendVal,
							endSep)[len(strings.Split(legendVal, endSep))-1]
				}

			} else {
				finalLegend = legendVal
			}

			customLogger("debug", "legend not empty, tags", tagsEmpty)
			customLogger("debug", "legend not empty, finalLegend", finalLegend)

			frame.Fields = append(frame.Fields,
				data.NewField(
					finalLegend,
					tags,
					[]float64{}).SetConfig(
					&data.FieldConfig{DisplayNameFromDS: finalLegend}),
			)
		}
		// iterate for time, value array and append it in current dataframe
		// by creating a temporary interface called rowVal and parsing it
		// properly
		for _, colName := range item.Values {
			rowVal := make([]interface{}, 2)

			var floatTime float64 = colName[0].(float64)
			var intTime int64 = int64(floatTime)
			var curVal string = colName[1].(string)

			rowVal[0] = time.Unix(intTime, 0)
			curValFinal, err := strconv.ParseFloat(curVal, 64)
			if err != nil {
				customLogger("error", "Failed to parse value", err)
				return frames, execTime, err
			}
			rowVal[1] = curValFinal
			customLogger("info", "rowvalue0", rowVal[0])
			customLogger("info", "rowvalue1", rowVal[1])
			//appending current key value pair in current dataframe
			frame.AppendRow(rowVal...)
			rowsTotal = rowsTotal + 1
		}
		//append current dataframe in global dataframe to be returned
		customLogger("info", "appending frame", "")
		frames = append(frames, frame)
		customLogger("info", "frame appended", "")
	}
	//return the final dataframe containing all the frames
	*rowsProcessed = rowsTotal
	customLogger("info", "dfreturn 2", execTime)
	customLogger("info", "dfreturn 3", err)
	customLogger("info", "dfreturn 1", "")
	return frames, execTime, err
}

// This is the query method which runs for each query present in current panel
// It is called by QueryData function for each query
func query(query backend.DataQuery, dbConn *sql.DB, deploymentType string) backend.DataResponse {
	response := backend.DataResponse{} //Response object to be returned
	// Unmarshal the JSON into our QueryModel and create a map of it.
	var err error
	var queryInterface interface{}
	errjson:= json.Unmarshal(query.JSON, &queryInterface)
	if errjson != nil {
		customLogger("error", "Json Unmarshal error", errjson)
		response.Error = errjson
		return response
	}
	queryDataMap := queryInterface.(map[string]interface{})
	//Now we have all or querydata in map queryDataMap.
	customLogger("info", "calling dumpstruct from", "query")
	dumpStruct(queryDataMap, "info")
	customLogger("debug", "Query object", query)
	customLogger("debug", "Query timerange From", query.TimeRange.From.Unix())
	customLogger("debug", "Query timerange To", query.TimeRange.To.Unix())

	//check the language type to set promql flag
	promql := true
	queryText := ""
	convertSqlResults := true
	prefetchsize := 100
	qryInputVal := ""
	legendTextVal := ""
	queryTextConverted := ""
	timeBeforeQuery := time.Now() //This is initialised to time before query
	timeAfterQuery := time.Now()  //This is set to the time when execution
	//any query finishes to calculate timers
	rowsProcessed := 0 //Rows processed for current query
	stepSize := "10"

	if queryDataMap["queryLang"] == "sql" {
		promql = false
		queryText = queryDataMap["exprSql"].(string)
		qryInputVal, _ = queryDataMap["exprSql"].(string)
		legendTextVal, _ = queryDataMap["legendFormatSql"].(string)
		customLogger("debug", "promql flg false value", promql)

		// get prefetch count from frontend and override the default if its not null
		customLogger("info", "Prefetch count fetched",
			queryDataMap["prefetchCountText"])
		customLogger("info", "Prefetch count default", prefetchsize)
		if prefetchcountobj, ok := queryDataMap["prefetchCountText"]; ok {
			if val, err := strconv.Atoi(prefetchcountobj.(string)); err == nil {
				prefetchsize = val
			}
		}
		customLogger("info", "Prefetch count final", prefetchsize)

		//check if convert sql is true to set convertSqlResults flag
		customLogger("debug", "queryDataMap[convsqlflg] value",
			queryDataMap["convertSqlResults"])

		if queryDataMap["convertSqlResults"] == false {
			convertSqlResults = false
			customLogger("debug", "make flg value true", convertSqlResults)
		} else {
			customLogger("debug", "keep flg value false", convertSqlResults)
		}

		if stepSizeObj, ok := queryDataMap["stepTextSql"]; ok {
			if _, err := strconv.Atoi(stepSizeObj.(string)); err == nil {
				stepSize = stepSizeObj.(string)
			}
		}

	} else {
		queryText = queryDataMap["exprProm"].(string)
		qryInputVal, _ = queryDataMap["exprProm"].(string)
		legendTextVal, _ = queryDataMap["legendFormatProm"].(string)
		customLogger("debug", "promql flg false value", promql)
		if stepSizeObj, ok := queryDataMap["stepTextProm"]; ok {
			if _, err := strconv.Atoi(stepSizeObj.(string)); err == nil {
				stepSize = stepSizeObj.(string)
			}
		}
	}

	var refString string
	refString, _ = queryDataMap["refId"].(string)
	customLogger("debug", "Query refString value", refString)

	//there can be different types of queries like promql , sql , metric find.
	// There are following conditions to handle them
	//This first if condition is for support of labels
	//when refString is "fetchLabels" we need to return the labels list
	//so that it can be used for autocompletion
	if refString == "fetchLabels" {
		queryText := ""
		if queryDataMap["timeFrom"] != nil && queryDataMap["timeTo"] != nil {
			fromTs := 0
			toTs := 0
			if fromTsObj, ok := queryDataMap["timeFrom"]; ok {
				if val, err := strconv.Atoi(fromTsObj.(string)); err == nil {
					fromTs = val / 1000
				}
			}
			if toTsObj, ok := queryDataMap["timeTo"]; ok {
				if val, err := strconv.Atoi(toTsObj.(string)); err == nil {
					toTs = val / 1000
				}
			}
			customLogger("debug", "FromTs to fetch labels case1", fromTs)
			customLogger("debug", "ToTs to fetch labels case1", toTs)
			queryText = fmt.Sprintf(getConstants("query_fetchlabels_str", deploymentType), fromTs,
				toTs)
		} else {
			secsToSub := int64(1 * 60 * 60) // hours*60(mins)*60(sec)
			customLogger("debug", "secs to subtract ", secsToSub)
			queryText = fmt.Sprintf(getConstants("query_fetchlabels_str", deploymentType),
				time.Now().Unix()-secsToSub,
				time.Now().Unix())
		}

		queryTextConverted = queryText
		logQueryInfo("Query to fetch labels", "Before", queryText)
		var rows *sql.Rows

		rows, err = dbConn.Query(queryText)
		
		if err != nil {
			customLogger("error", "My db rows error1", err)
			response.Error = err
			return response
		} else {
			defer rows.Close()
			customLogger("info", "labels query executed", "")
			frame, err := getSingleColumnDF(rows, &rowsProcessed, &timeAfterQuery)
			if err != nil {
				return response
			}

			//log query with timings
			logQueryStatsInfo("Query to fetch labels", "After", queryText, rowsProcessed, timeBeforeQuery, timeAfterQuery)
			response.Frames = frame
			return response
		}
	}

	var rows *sql.Rows

	// there can be different types of queries like promql , sql , metric find.
	// There are following conditions to handle them This first if condition is
	// for support of query variable when refString is "metricFindQuery" we
	// need to return the tags for the corresponding metric so that it can be
	// used in the list for query variables
	if refString == "metricFindQuery" {
		// section to return tags for query variables
		var metricName string
		metricName, _ = queryDataMap["expr"].(string)
		queryLang, _ := queryDataMap["queryLang"].(string)
		customLogger("info", "Querylang for metricfindquery", queryLang)
		//follwing query fetches the tags for given metric name
		mn := metricName[:strings.Index(metricName, "&start")]
		start := metricName[strings.Index(metricName, "start")+6 : strings.Index(metricName, "start")+16]
		end := metricName[strings.Index(metricName, "end")+4 : strings.Index(metricName, "end")+14]

		queryText = fmt.Sprintf(getConstants("variable_query_str", deploymentType), mn, start, end)
		queryTextConverted = queryText
		customLogger("debug", "Fetching Tags for metric", metricName)

		logQueryInfo("Query to Fetch Tags", "Before", queryText)

		rows, err = dbConn.Query(queryText)
		
		if err != nil {
			customLogger("error", "My db rows error2", err)
			response.Error = err
			return response
		}
		defer rows.Close()
		// function to convert result of query(single column of tags) to
		// dataframe
		frame, err := getSingleColumnDF(rows, &rowsProcessed, &timeAfterQuery)
		if err != nil {
			response.Error = err
			return response
		}

		//log query with timings
		logQueryStatsInfo("Query to Fetch Tags", "After", queryText, rowsProcessed, timeBeforeQuery, timeAfterQuery)
		//append frame in response and return
		response.Frames = frame
		//customLogger("debug", "My query returned response", response)
		return response

	} else if refString == "getKeysForAdHocFilter" {
		//follwing query fetches the keys for adhoc filter
		queryText = getConstants("adhock_key_query_str", deploymentType)
		queryTextConverted = queryText
		logQueryInfo("Query to Fetch keys for adhocfilter", "Before", queryText)

		rows, err = dbConn.Query(queryText)

		if err != nil {
			customLogger("error", "My db rows error3", err)
			response.Error = err
			return response
		}
		defer rows.Close()

		frame, err := getSingleColumnDF(rows, &rowsProcessed, &timeAfterQuery)
		if err != nil {
			response.Error = err
			return response
		}

		//log query with timings
		logQueryStatsInfo("Query to Fetch keys for adhocfilter", "After",
			queryText, rowsProcessed, timeBeforeQuery, timeAfterQuery)

		//append frame in response and return
		response.Frames = frame
		customLogger("debug", "My query returned response", response)
		return response

	} else if refString == "getValueforKeyAdHocFilter" {
		//follwing query fetches the values corresponding key for adhoc filter
		queryText = fmt.Sprintf(getConstants("adhock_value_query_str", deploymentType),
			queryDataMap["rawQueryText"].(string))
		queryTextConverted = queryText
		logQueryInfo("Query to Fetch value for key of adhocfilter:", "Before",
			queryText)

		rows, err = dbConn.Query(queryText)

		if err != nil {
			customLogger("debug", "My db rows error4", err)
			response.Error = err
			return response
		}
		defer rows.Close()

		// function to convert result of query(single column of tags) to
		// dataframe
		frame, err := getSingleColumnDF(rows, &rowsProcessed, &timeAfterQuery)
		if err != nil {
			response.Error = err
			return response
		}

		//log query with timings
		logQueryStatsInfo("Query to Fetch value for key of adhocfilter:",
			"After", queryText, rowsProcessed, timeBeforeQuery, timeAfterQuery)

		//append frame in response and return
		response.Frames = frame
		customLogger("debug", "My query returned response", response)
		return response

	} else if promql {
		//This if condition is when language type specified is promql
		//first we need to convert the promql in required sql format
		customLogger("debug", "Language type is Promql, promql flg", promql)
		customLogger("debug", "queryDataMap value", queryDataMap)

		promqlToSql, err := getPromQLToSQL(query.TimeRange.From,
			query.TimeRange.To,
			queryText, stepSize,
			deploymentType)
		if err != nil {
			response.Error = err
			return response
		}

		logQueryInfo("Converted Query to fire:", "Before", promqlToSql)

		rows, err = dbConn.Query(promqlToSql)
		if err != nil {
			customLogger("error", "My db rows error5", err)
			response.Error = err
			return response
		}
		defer rows.Close()
		queryText = promqlToSql
	} else {
		//This if condition is when language type specified is SQL. Here we
		//donot need to convert the query here as sql can be directly executed
		customLogger("debug", "Language type is Sql, promql flag", promql)
		customLogger("debug", "My qry in SQL", queryText)

		step, _ := strconv.ParseInt(stepSize, 10, 64)
		// change queries to support the macros of grafana's oracle plugin
		// 1. If query contains $__timefilter(), replace it with greater
		//    and less than selected time range's timestamp
		if strings.Contains(queryText, "$__timeFilter") {
			indx1 := strings.Index(queryText, "$__timeFilter")
			indx2 := strings.Index(queryText[indx1:], ")")
			val := queryText[indx1+14 : indx1+indx2]
			queryPart1 := queryText[:indx1]
			queryPart2 := queryText[indx1+indx2+1:]
			fStr := queryPart1 + " " + val +
				">= to_date('19700101', 'YYYYMMDD') +" +
				" ( 1 / 24 / 60 / 60 ) * :start_time and " + val +
				"<= to_date('19700101', 'YYYYMMDD') +" +
				" ( 1 / 24 / 60 / 60 ) * :end_time " + queryPart2
			queryText = fStr
			// The above manipulation will replace '$__timefiler(ts)' in query
			// with 'ts >= to_date(selected_start_date in grafana) and
			// ts <= to_date(selected_end_time in grafana)'.
			customLogger("debug",
				"Query contains $__timeFilter, changed query in sql", fStr)
		}

		// 2. If query contains $__unixEpochFilter() macro, replace it with
		//    greater and less than selected time range's epoch seconds. The
		//    difference in $__timefilter() and $__unixEpochFilter() is the
		//    prior one uses date to bound query, while later uses unix epochs
		if strings.Contains(queryText, "$__unixEpochFilter") {
			indx1 := strings.Index(queryText, "$__unixEpochFilter")
			indx2 := strings.Index(queryText[indx1:], ")")
			val := queryText[indx1+19 : indx1+indx2]
			queryPart1 := queryText[:indx1]
			queryPart2 := queryText[indx1+indx2+1:]
			fStr := queryPart1 + " " + val + ">= :start_time*1000 and " +
				val + "<= :end_time*1000 " + queryPart2
			queryText = fStr
			customLogger("debug",
				"Query contains $__unixEpochFilter, changed query in sql", fStr)
		}

		// 3. If query contains $__timeGroup() , change the text such that it
		//    supports group by. Also, timeGroup may be present in query
		//    multiple times(in select and in groupby clause) so using a for
		//    loop to execute this till $__timeGroup is present in the query.
		for strings.Contains(queryText, "$__timeGroup") {
			// The timegroup macro will be represented as
			// $__timeGroup(dateColumn,5m) in query, thus first thing is to
			// extract the parameters of the macro(columnname and duration).
			indx1 := strings.Index(queryText, "$__timeGroup")
			indx2 := strings.Index(queryText[indx1:], ")")
			val := queryText[indx1+13 : indx1+indx2]
			queryPart1 := queryText[:indx1]
			queryPart2 := queryText[indx1+indx2+1:]
			// here queryPart1 is initial part of query, value is the parameters
			// of macro and queryPart2 is remaining part of query.

			// Now we need columnname and duration for grouping time. For that
			// we need to split the val string to obtain first and second
			// parameter of $__timeGroup.
			splits := strings.Split(val, ",")
			split1 := strings.TrimSpace(splits[0])
			split2 := strings.TrimSpace(splits[1])

			// If $__interval is used as duration then we use the stepSize
			// as its value.Else convert whatever time is given in mins, secs ,
			// hours etc to seconds.
			if split2 == "$__interval" {
				split2 = strconv.FormatInt(step, 10)
			} else if strings.HasSuffix(split2, "s") {
				split2 = split2[:len(split2)-1]
			} else if strings.HasSuffix(split2, "m") {
				minval, _ := strconv.ParseInt(split2[:len(split2)-1], 10, 64)
				secsval := minval * 60
				split2 = strconv.FormatInt(secsval, 10)
			} else if strings.HasSuffix(split2, "h") {
				hrval, _ := strconv.ParseInt(split2[:len(split2)-1], 10, 64)
				minval := hrval * 60
				secsval := minval * 60
				split2 = strconv.FormatInt(secsval, 10)
			}

			// Finally creating the clause to be used in groupby.
			// Here we need to replace '$__timeGroup(timecolumn, interval)' by
			// TO_DATE('19700101', 'YYYYMMDD') + ( 1 / 24 / 60 / 60 / 1000) *
			// FLOOR((timecolumn - TO_TIMESTAMP('1970-01-01 00:00:00',
			// 'yyyy-mm-dd hh24:mi:ss') +
			// TO_DATE ('1970-01-01 00:00:00', 'YYYY-mm-dd HH24:MI:SS')
			// - TO_DATE ('1970-01-01 00:00:00', 'YYYY-mm-dd HH24:MI:SS')
			// )*24*60*60*1000/interval/1000)*interval*1000
			// The above logic of groupby is taken from Grafana's oracle plugin
			// to make our plugin behave same as theirs for this macro while
			// migrating any grafana's oracle plugin to our plugin
			groupStr := "TO_DATE('19700101', 'YYYYMMDD') +" +
				" ( 1 / 24 / 60 / 60 / 1000) * FLOOR((" +
				split1 +
				" - TO_TIMESTAMP('1970-01-01 00:00:00'," +
				"'yyyy-mm-dd hh24:mi:ss') + " +
				"TO_DATE ('1970-01-01 00:00:00', " +
				"'YYYY-mm-dd HH24:MI:SS') " +
				"- TO_DATE ('1970-01-01 00:00:00'," +
				" 'YYYY-mm-dd HH24:MI:SS'))*24*60*60*1000/" +
				split2 +
				"/1000)*" +
				split2 +
				"*1000"
			fStr := queryPart1 + " " + groupStr + queryPart2
			queryText = fStr
			customLogger("debug",
				"Query contains $__timeGroup, changed query in sql", fStr)
		}

		// 3. If query contains '$__time(t1)' replace it with 't1 as time'
		if strings.Contains(queryText, "$__time") {
			indx1 := strings.Index(queryText, "$__time")
			indx2 := strings.Index(queryText[indx1:], ")")
			val := queryText[indx1+8 : indx1+indx2]
			queryPart1 := queryText[:indx1]
			queryPart2 := queryText[indx1+indx2+1:]
			fStr := queryPart1 + " " + val + " as time " + queryPart2
			customLogger("debug",
				"Query contains $__time, changed query in sql", fStr)
			queryText = fStr
		}

		//change query to add timestamp
		if strings.Contains(queryText, ":start_time") &&
			strings.Contains(queryText, ":end_time") {
			//both start and end time found
			stVal := fmt.Sprintf("%d", query.TimeRange.From.Unix())
			etVal := fmt.Sprintf("%d", query.TimeRange.To.Unix())
			queryText2 := strings.Replace(queryText, ":start_time", stVal, -1)
			queryText = queryText2
			queryText3 := strings.Replace(queryText, ":end_time", etVal, -1)
			queryText = queryText3
			customLogger("debug", "Changed qry in SQL case1, sql_1", queryText)
		} else if strings.Contains(queryText, ":start_time") {
			//only start time found
			stVal := fmt.Sprintf("%d", query.TimeRange.From.Unix())
			queryText2 := strings.Replace(queryText, ":start_time", stVal, -1)
			queryText = queryText2
			customLogger("debug", "Changed qry in SQL case2, sql_2", queryText)
		} else if strings.Contains(queryText, ":end_time") {
			//only end time found
			etVal := fmt.Sprintf("%d", query.TimeRange.To.Unix())
			queryText3 := strings.Replace(queryText, ":end_time", etVal, -1)
			queryText = queryText3
			customLogger("debug", "Changed qry in SQL case3, sql_3", queryText)
		}

		queryTextConverted = queryText
		logQueryInfo("Final sql query before translation is :", "Before", queryText)

		logQueryInfo("Final sql query after translation is :", "Before", queryText)
		//execute the query and store results in rows
		rows, err = dbConn.Query(queryText, godror.FetchRowCount(prefetchsize))

		if err != nil {
			customLogger("error", "My db rows error6", err)
			response.Error = err
			return response
		}
		defer rows.Close()
	}
	customLogger("debug", "My db rows success", rows)
	defer rows.Close()

	frames, execTime, err := getDataFrameFromRows(
		rows,
		promql,
		convertSqlResults,
		qryInputVal,
		legendTextVal,
		queryTextConverted,
		&rowsProcessed,
		&timeAfterQuery)
	if err != nil {
		customLogger("error", "Errong getting output", err.Error())
		response.Error = err
		return response
	}
	customLogger("info", "no error", "")

	var rttCommon time.Duration
	if execTime == "" {
		customLogger("debug", "Exectime not found, exectime=", execTime)
		customLogger("info", "no error2", "")
	} else {
		customLogger("debug", "Exectime found, exectime=", execTime)
		timedur := execTime + "s"
		execTimeDurn, _ := time.ParseDuration(timedur)
		rttCommon = timeAfterQuery.Sub(timeBeforeQuery)
		networkTime := (rttCommon - execTimeDurn).String()
		customLogger("debug", "Network time, networkTime=", networkTime)
		customLogger("info", "no error3", "")
	}

	//log query with timings
	customLogger("info", "no error5", "")
	logQueryStatsInfo("Query Final Executed:", "After", queryText, rowsProcessed, timeBeforeQuery, timeAfterQuery)

	if response.Error != nil {
		return response
	}
	customLogger("info", "no error6", "")
	response.Frames = frames
	customLogger("info", "no error 7", "")
	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (jd *OracleDatasource) CheckHealth(
	_ context.Context,
	req *backend.CheckHealthRequest) (
	*backend.CheckHealthResult, error) {
	customLogger("debug", "CheckHealth called with request", req)
	customLogger("info", "taken data from global object", "")

	customLogger("info", "calling dumpstruct from", "checkhealth")
	dumpStruct(*jd, "info")
	Auth := jd.QueryAuth
	if Auth == "BASIC" {
		connString := fmt.Sprintf("%s/%s@%s:%s/%s", jd.DbUser, jd.secureCredData.DecryptedSecureJSONData["dbPassword"], jd.DbHostName, jd.DbPortName, jd.DbServiceName)
		var status = backend.HealthStatusOk
		var message = "Data source is working"

		//try to open a connection with db
		db, err := GetSqlDBWithGoDror(connString)
		if err != nil {
			//if error opening connection, change the message and status to
			//error
			customLogger("error", "My db connect error", err)
			status = backend.HealthStatusError
			message = "Error Connecting to Database!!! ERROR: " + err.Error()
		} else {
			customLogger("info", "My db connection success", "")
		}
		defer db.Close()

		//return the message of healthcheck
		return &backend.CheckHealthResult{
			Status:  status,
			Message: message,
		}, nil

	} else {
		connString := jd.DbUser + "/" + jd.secureCredData.DecryptedSecureJSONData["dbPassword"] + "@" + jd.DbConnectString
		var status = backend.HealthStatusOk
		var message = "Data source is working"

		//try to open a connection with db
		db, err := GetSqlDBWithGoDror(connString)
		if err != nil {
			//if error opening connection, change the message and status to
			//error
			customLogger("error", "My db connect error", err)
			status = backend.HealthStatusError
			message = "Error Connecting to Database!!! ERROR: " + err.Error()
		} else {
			customLogger("info", "My db connection success", "")
		}
		defer db.Close()
		//return the message of healthcheck
		return &backend.CheckHealthResult{
			Status:  status,
			Message: message,
		}, nil
	}
}

// SubscribeStream is called when a client wants to connect to a stream. This
// callback allows sending the first message.
func (d *OracleDatasource) SubscribeStream(
	_ context.Context,
	req *backend.SubscribeStreamRequest) (
	*backend.SubscribeStreamResponse,
	error) {
	customLogger("info", "SubscribeStream called with request", req)

	status := backend.SubscribeStreamStatusPermissionDenied
	if req.Path == "stream" {
		// Allow subscribing only on expected path.
		status = backend.SubscribeStreamStatusOK
	}
	return &backend.SubscribeStreamResponse{
		Status: status,
	}, nil
}

// RunStream is called once for any open channel.  Results are shared with
// everyone subscribed to the same channel.
func (d *OracleDatasource) RunStream(
	ctx context.Context,
	req *backend.RunStreamRequest,
	sender *backend.StreamSender) error {
	customLogger("debug", "RunStream called with request", req)

	// Create the same data frame as for query data.
	frame := data.NewFrame("response")

	// Add fields (matching the same schema used in QueryData).
	frame.Fields = append(frame.Fields,
		data.NewField("time", nil, make([]time.Time, 1)),
		data.NewField("values", nil, make([]int64, 1)),
	)

	counter := 0

	// Stream data frames periodically till stream closed by Grafana.
	for {
		select {
		case <-ctx.Done():
			customLogger("debug",
				"Context done, finish streaming with path", req.Path)
			return nil
		case <-time.After(time.Second):
			// Send new data periodically.
			frame.Fields[0].Set(0, time.Now())
			frame.Fields[1].Set(0, int64(10*(counter%2+1)))

			counter++

			err := sender.SendFrame(frame, data.IncludeAll)
			if err != nil {
				log.DefaultLogger.Error("Error sending frame", "error", err)
				continue
			}
		}
	}
}

// PublishStream is called when a client sends a message to the stream.
func (d *OracleDatasource) PublishStream(
	_ context.Context,
	req *backend.PublishStreamRequest) (
	*backend.PublishStreamResponse,
	error) {
	customLogger("debug", "PublishStream called with request", req)

	// Do not allow publishing at all.
	return &backend.PublishStreamResponse{
		Status: backend.PublishStreamStatusPermissionDenied,
	}, nil
}
