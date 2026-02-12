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

import { addLabelToQuery, addLabelToSelector } from '../AddLabelToQuery';

describe('addLabelToSelector', () => {
  it('adds label to empty selector', () => {
    const result = addLabelToSelector('', 'job', 'api');
    expect(result).toBe('{job="api"}');
  });

  it('adds label to existing selector', () => {
    const result = addLabelToSelector('env="prod"', 'job', 'api');
    expect(result).toBe('{env="prod",job="api"}');
  });

  it('uses provided operator', () => {
    const result = addLabelToSelector('', 'status', '5..', '=~');
    expect(result).toBe('{status=~"5.."}');
  });

  it('does not duplicate identical labels', () => {
    const result = addLabelToSelector('job="api"', 'job', 'api');
    expect(result).toBe('{job="api"}');
  });
});

describe('addLabelToQuery', () => {
  it('throws if label key is missing', () => {
    expect(() => addLabelToQuery('metric', '', 'v')).toThrow();
  });

  it('adds selector to bare metric', () => {
    const result = addLabelToQuery('http_requests_total', 'job', 'api');

    expect(result).toBe('http_requests_total{job="api"}');
  });

  it('adds label to existing selector', () => {
    const result = addLabelToQuery('http_requests_total{method="GET"}', 'job', 'api');

    expect(result).toBe('http_requests_total{job="api",method="GET"}');
  });

  it('does not modify Grafana variables', () => {
    const result = addLabelToQuery('${__rate_interval}', 'job', 'api');

    expect(result).toBe('${__rate_interval}');
  });

  it('converts Infinity to +Inf', () => {
    const result = addLabelToQuery('metric', 'le', Infinity);

    expect(result).toBe('metric{le="+Inf"}');
  });

  it('adds label to multiple metrics', () => {
    const result = addLabelToQuery('sum(rate(http_requests_total[5m]))', 'job', 'api');

    expect(result).toContain('{job="api"}');
  });
});
