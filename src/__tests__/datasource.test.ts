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

jest.mock('@grafana/runtime', () => ({
  getTemplateSrv: jest.fn(() => ({
    replace: jest.fn((v: string) => v),
    getAdhocFilters: jest.fn(() => []),
  })),

  DataSourceWithBackend: class {
    name = 'test-datasource';

    query() {
      return {
        toPromise: jest.fn().mockResolvedValue({
          data: [],
        }),
      };
    }
  },
}));

const mockQueryResponse = (bufferValue: any) => ({
  toPromise: jest.fn().mockResolvedValue({
    data: [
      {
        fields: [
          {
            values: {
              buffer: [bufferValue],
            },
          },
        ],
      },
    ],
  }),
});

import { DataSource } from '../datasource';

describe('DataSource', () => {
  it('constructs without crashing', () => {
    const ds = new DataSource({} as any);
    expect(ds).toBeDefined();
  });
});

it('isExist returns value when function succeeds', () => {
  const ds = new DataSource({} as any);
  const result = ds.isExist(() => 'ok');
  expect(result).toBe('ok');
});

it('isExist returns false when function throws', () => {
  const ds = new DataSource({} as any);
  const result = ds.isExist(() => {
    throw new Error('fail');
  });
  expect(result).toBe(false);
});

it('serializeVariableValue returns string as-is', () => {
  const ds = new DataSource({} as any);
  const result = (ds as any).serializeVariableValue('abc');
  expect(result).toBe('abc');
});

it('serializeVariableValue returns empty string for empty array', () => {
  const ds = new DataSource({} as any);
  const result = (ds as any).serializeVariableValue([]);
  expect(result).toBe('');
});

it('serializeVariableValue joins array values with |', () => {
  const ds = new DataSource({} as any);
  const result = (ds as any).serializeVariableValue(['a', 'b', 'c']);
  expect(result).toBe('a|b|c');
});

it('prometheusRegularEscape escapes backslashes and quotes', () => {
  const ds = new DataSource({} as any);
  const result = ds.prometheusRegularEscape(`a\\b'c`);
  expect(result).toBe(`a\\\\b\\\\'c`);
});

it('prometheusRegularEscape returns non-string as-is', () => {
  const ds = new DataSource({} as any);
  expect(ds.prometheusRegularEscape(123)).toBe(123);
});

it('applyAdhocFilters applies label filters', () => {
  const ds = new DataSource({} as any);

  const result = (ds as any).applyAdhocFilters('metric', [{ key: 'job', operator: '=', value: 'node' }]);

  expect(result).toContain('job');
});

it('applyAdhocFilters escapes regex operators', () => {
  const ds = new DataSource({} as any);

  const result = (ds as any).applyAdhocFilters('metric', [{ key: 'job', operator: '=~', value: `a'b` }]);

  expect(result).toContain(`\\\\'`);
});

it('applyTemplateVariables replaces expressions safely', () => {
  const ds = new DataSource({} as any);

  const query = {
    expr: 'metric',
    exprSql: 'metric_sql',
    exprProm: 'metric_prom',
  } as any;

  const result = ds.applyTemplateVariables(query);

  expect(result.expr).toBe('metric');
  expect(result.exprSql).toBe('metric_sql');
  expect(result.exprProm).toBe('metric_prom');
});

it('getTagKeys returns metric keys', async () => {
  const ds = new DataSource({} as any);

  jest.spyOn(ds, 'query').mockReturnValue(mockQueryResponse(JSON.stringify({ data: ['job', 'instance'] })) as any);

  const result = await ds.getTagKeys();

  expect(result).toEqual([{ text: 'job' }, { text: 'instance' }]);
});

it('getTagKeys returns empty array if buffer is not string', async () => {
  const ds = new DataSource({} as any);

  jest.spyOn(ds, 'query').mockReturnValue(mockQueryResponse(123) as any);

  const result = await ds.getTagKeys();
  expect(result).toEqual([]);
});

it('getTagValues returns values for key', async () => {
  const ds = new DataSource({} as any);

  jest.spyOn(ds, 'query').mockReturnValue(mockQueryResponse(JSON.stringify({ data: ['us-east', 'us-west'] })) as any);

  const result = await ds.getTagValues({ key: 'region' });

  expect(result).toEqual([{ text: 'us-east' }, { text: 'us-west' }]);
});

it('metricFindQuery returns formatted metric names', async () => {
  const ds = new DataSource({} as any);

  jest.spyOn(ds, 'fetchMetricNames').mockResolvedValue({
    data: [
      {
        fields: [
          {
            values: {
              buffer: [
                JSON.stringify({
                  data: [{ __name__: 'up', job: 'node', instance: 'localhost' }],
                }),
              ],
            },
          },
        ],
      },
    ],
  } as any);

  const result = await ds.metricFindQuery({
    query: 'up',
    queryLang: 'promql',
  } as any);

  expect(result[0].text).toContain('up');
  expect(result[0].text).toContain('job="node"');
  expect(result[0].text).toContain('instance="localhost"');
});

it('fetchStaticLabels returns label list', async () => {
  const ds = new DataSource({} as any);

  jest.spyOn(ds, 'getFromStr').mockReturnValue('123456');
  jest.spyOn(ds, 'getToStr').mockReturnValue('123999');

  jest.spyOn(ds, 'query').mockReturnValue(mockQueryResponse(JSON.stringify({ data: ['cpu', 'memory'] })) as any);

  const result = await ds.fetchStaticLabels();

  expect(result).toEqual([{ name: 'cpu' }, { name: 'memory' }]);
});

/*it('fetchStaticLabels returns empty array on error', async () => {
  const ds = new DataSource({} as any);

  jest.spyOn(ds, 'getFromStr').mockReturnValue('123');
  jest.spyOn(ds, 'getToStr').mockReturnValue('456');

  jest.spyOn(ds, 'query').mockImplementation(() => {
    throw new Error('fail');
  });

  const result = await ds.fetchStaticLabels();
  expect(result).toEqual([]);
});*/
it('fetchStaticLabels returns empty array on error', async () => {
  const ds = new DataSource({} as any);

  jest.spyOn(ds, 'getFromStr').mockReturnValue('123');
  jest.spyOn(ds, 'getToStr').mockReturnValue('456');

  jest.spyOn(ds, 'query').mockImplementation(() => {
    throw new Error('fail');
  });

  const consoleSpy = jest.spyOn(console, 'error').mockImplementation();

  const result = await ds.fetchStaticLabels();
  expect(result).toEqual([]);

  consoleSpy.mockRestore();
});
