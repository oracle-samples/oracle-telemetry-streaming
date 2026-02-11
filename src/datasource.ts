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



import addLabelToQuery from "./AddLabelToQuery";

import { MetricFindValue, DataSourceInstanceSettings } from "@grafana/data";
//For providing support of query variable we need to import MetricFindValue

import {
  DataSourceOptionsObj,
  QueryObj,
  VariableQueryObject,
  InData,
} from "./types";
//For providing support of query variable we need to import VariableQueryObject

import { getTemplateSrv, DataSourceWithBackend } from "@grafana/runtime";
//For providing support of custom variable we need to import getTemplateSrv

export class DataSource extends DataSourceWithBackend<
  QueryObj,
  DataSourceOptionsObj
> {
  constructor(
    instanceSettings: DataSourceInstanceSettings<DataSourceOptionsObj>
  ) {
    super(instanceSettings);
  }

  //get from time value from selected range
  getFromStr() {
    const templateSrv = getTemplateSrv();
    const strret = templateSrv.replace("$__from", {}, (variables: any) => {
      return variables;
    });
    return strret;
  }

  extractValue(response: any) {
    return (
      response?.data?.[0]?.fields?.[0]?.values?.get?.(0) ??
      response?.data?.[0]?.fields?.[0]?.values?.buffer?.[0] ??
      response?.data?.[0]?.fields?.[0]?.values?.[0]
    );
  }

  //get to time value from selected range
  getToStr() {
    const templateSrv = getTemplateSrv();
    const strret = templateSrv.replace("$__to", {}, (variables: any) => {
      return variables;
    });
    return strret;
  }

  //get value from nested object if exist else return undefined
  isExist(arg: any) {
    try {
      return arg();
    } catch (e) {
      return false;
    }
  }

  // async fetchStaticLabels(): Promise<InData[]> {
  //   console.log('shanu log');
  // return Promise.resolve([
  //   { name: "cpu_usage" },
  //   { name: "cpu_usage2" },
  //   { name: "cpu_usage3" },
  //   { name: "memory_usage" },
  //   { name: "disk_reads" },
  //   { name: "http_requests_total" },
  //   { name: "uptime_seconds" },
  // ]);
  // }

  async fetchStaticLabels(): Promise<InData[]> {
    const fromms = this.getFromStr();
    const toms = this.getToStr();

    if (!fromms || !toms) {
      return [];
    }

    try {
      const response = await this.query({
        targets: [
          {
            refId: "fetchLabels",
            rawQueryText: "",
            expr: "",
            exprProm: "",
            exprSql: "",
            timeFrom: fromms,
            timeTo: toms,
          },
        ],
      } as any).toPromise();

      const frame = response?.data?.[0];
      if (!frame || !frame.fields?.[0]) {
        return [];
      }

      const buffer = this.extractValue(response);
      if (typeof buffer !== "string") {
        return [];
      }

      const parsed = JSON.parse(buffer);
      const arr_tags = parsed?.data;
      if (!Array.isArray(arr_tags)) {
        return [];
      }

      // Now map the array items.
      return arr_tags.map((name: string) => ({ name }));
    } catch (err) {
      console.error("fetchStaticLabels failed:", err);
      return [];
    }
  }

  //This method is called whenever we change the custom variable in
  //the grafana panel. It gets the variables details through
  //getTemplateSrv() and uses its replace method to replace occurances
  //of the defined variables in current query.Also this function is called
  //at time of loading plugin so we fetch labels with this functions help
  applyTemplateVariables(query: QueryObj) {
    //this part is to load labels in cache initially
    const templateSrv = getTemplateSrv();
    const adhocFilters = (getTemplateSrv() as any).getAdhocFilters(this.name);

    const applyTemplate = (value?: string) =>
      value
        ? templateSrv.replace(value, {}, (variables: any) =>
            this.serializeVariableValue(variables)
          )
        : "";

    const nextQuery: QueryObj = {
      ...query,
      expr: applyTemplate(query.expr),
      exprSql: applyTemplate(
        this.applyAdhocFilters(query.exprSql, adhocFilters)
      ),
      exprProm: applyTemplate(
        this.applyAdhocFilters(query.exprProm, adhocFilters)
      ),
    };

    return nextQuery;
  }

  private serializeVariableValue(variables: any): string {
    if (typeof variables === "string") {
      return variables;
    }

    if (!Array.isArray(variables) || variables.length === 0) {
      return "";
    }

    return variables.join("|");
  }

  async getTagKeys() {
    const values: MetricFindValue[] = [];
    //calling this api to fetch the keys
    const response = await this.query({
      targets: [
        {
          refId: "getKeysForAdHocFilter",
          rawQueryText: "",
          expr: "",
          exprProm: "",
          exprSql: "",
          timeColumns: [],
        },
      ],
    } as any).toPromise();

    if (response?.error) {
      throw new Error(response.error.message);
    }
    const buffer = this.extractValue(response);
    if (typeof buffer !== "string") {
      return values;
    }

    const parsed = JSON.parse(buffer);
    const arr_tags = parsed?.data;
    if (!Array.isArray(arr_tags)) {
      return values;
    }

    for (let i = 0; i < arr_tags.length; i += 1) {
      values.push({ text: arr_tags[i] });
    }

    return values;
  }

  async getTagValues(options: any = {}) {
    const values: MetricFindValue[] = [];
    const response = await this.query({
      targets: [
        {
          refId: "getValueforKeyAdHocFilter",
          rawQueryText: options.key,
          expr: "",
          exprSql: "",
          exprProm: "",
          timeColumns: [],
        },
      ],
    } as any).toPromise();

    if (response?.error) {
      throw new Error(response.error.message);
    }

    const buffer = this.extractValue(response);
    if (typeof buffer !== "string") {
      return values;
    }

    const parsed = JSON.parse(buffer);
    const arr_tags = parsed?.data;
    if (!Array.isArray(arr_tags)) {
      return values;
    }

    for (let i = 0; i < arr_tags.length; i += 1) {
      values.push({ text: arr_tags[i] });
    }
    return values;
  }

  prometheusRegularEscape(value: any) {
    return typeof value === "string"
      ? value.replace(/\\/g, "\\\\").replace(/'/g, "\\\\'")
      : value;
  }

  // this method fetches the tags corresponding to the promql query given for
  // query variable.  In backend we check the refId of query and if its
  //'metricFindQuery' we simply return tags for selected promql query
  async fetchMetricNames(query: string, queryLanguage: string) {
    let fromms = this.getFromStr();
    let toms = this.getToStr();
    // changing millisecs epoch to secs epoch by dropping last 3 characters
    let fromstr = fromms.substring(0, fromms.length - 3);
    let tostr = toms.substring(0, toms.length - 3);
    const query_with_time = query + "&start=" + fromstr + "&end=" + tostr;
    const response = await this.query({
      targets: [
        {
          refId: "metricFindQuery",
          rawQueryText: query_with_time,
          queryLang: queryLanguage,
          expr: query_with_time,
          timeColumns: [],
        },
      ],
    } as any).toPromise();

    if (response?.error) {
      throw new Error(response.error.message);
    }
    return response;
  }
  //this method first calls fetchMetricNames to get tags for the promql query
  //that is used for query variable support and then it does some manipulation
  //and makes array called values of type MetricFindValue which is
  //returned and rendered as options list in query variable in grafana.
  async metricFindQuery(
    query: VariableQueryObject,
    options?: any
  ): Promise<MetricFindValue[]> {
    // Retrieve DataQueryResponse based on query.
    const response = await this.fetchMetricNames(query.query, query.queryLang);
    const values: MetricFindValue[] = [];
    //values.push({ text: 'up{instance="slc.us.oracle.com:9100",job="node"}' });
    //    var arr_tags = JSON.parse(response['data'][0].fields[0].values.buffer[0]).result_output.data;
    const buffer = this.extractValue(response);
    if (typeof buffer !== "string") {
      return values;
    }

    const parsed = JSON.parse(buffer);
    const arr_tags = parsed?.data;
    if (!Array.isArray(arr_tags)) {
      return values;
    }

    for (const item of arr_tags) {
      const metric = item.__name__;

      const parts = Object.entries(item).map(([k, v]) => `${k}="${v}"`);
      const joined = `{${parts.join(",")}}`;
      values.push({ text: `${metric}${joined}` });
    }
    return values;
  }

  private applyAdhocFilters(
    expr: string | undefined,
    filters: Array<{ key?: any; operator?: any; value?: any }>
  ) {
    return filters.reduce((acc: string, filter) => {
      const { key, operator } = filter;
      let { value } = filter;
      if (operator === "=~" || operator === "!~") {
        value = this.prometheusRegularEscape(value);
      }
      return addLabelToQuery(acc, key, value, operator);
    }, expr ?? "");
  }
}
