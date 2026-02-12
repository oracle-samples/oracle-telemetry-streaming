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

import { DataQuery, DataSourceJsonData } from '@grafana/data';

//This is the interface for query which contains all the fiels required with
//query that are sent from frontend to backend
export interface QueryObj extends DataQuery {
  //fields promql
  exprProm?: string;
  legendFormatProm?: string;
  stepTextProm?: string;
  //fields sql
  exprSql?: string;
  legendFormatSql?: string;
  stepTextSql?: string;
  prefetchCountText?: string;
  convertSqlResults?: boolean;
  //common fields
  timeRange?: string;
  queryLang?: string;
  expr?: string;
  pointsFillSecs?: string;
}

export const defaultQuery: Partial<QueryObj> = {};

/**
 * These are options configured for each DataSource instance
 */
export interface DataSourceOptionsObj extends DataSourceJsonData {
  dbHostName: string;
  dbPortName: string;
  dbServiceName: string;
  queryAuth?: string;
  deploymentType?: string;
  //for db datasource type
  dbUser?: string;
  dbConnectString?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface SecureJsonData {
  //secure fields for db datasource type
  dbPassword?: string;
}
// for query variable we created the following interface
export interface VariableQueryObject {
  query: string;
  queryLang: string;
}

export interface InData {
  name: string;
}
