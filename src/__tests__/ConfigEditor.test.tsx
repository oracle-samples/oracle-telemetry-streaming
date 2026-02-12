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

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { ConfigEditor } from '../ConfigEditor';

jest.mock('@grafana/ui', () => {
  const actual = jest.requireActual('@grafana/ui');

  return {
    ...actual,
    Select: ({ onChange, options, value }: any) => (
      <select
        data-testid="mock-select"
        value={value?.value}
        onChange={(e) => onChange(options.find((o: any) => o.value === e.target.value))}
      >
        {options.map((o: any) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    ),
  };
});

function setup(overrides?: any) {
  const onOptionsChange = jest.fn();

  const options = {
    id: 1,
    uid: 'oracle-telemetry-test',
    orgId: 1,
    name: 'Oracle Telemetry',
    type: 'oracle-oracle-telemetry',
    access: 'proxy',
    url: '',
    jsonData: {
      queryAuth: 'TNS',
      deploymentType: 'ON-PREM',
      ...(overrides?.jsonData ?? {}),
    },
    secureJsonFields: {},
    secureJsonData: {},
    ...overrides,
  };

  render(<ConfigEditor options={options as any} onOptionsChange={onOptionsChange} />);

  return { onOptionsChange };
}

describe('ConfigEditor', () => {
  it('renders base fields', () => {
    setup();

    expect(screen.getByText('Connection Type')).toBeInTheDocument();
    expect(screen.getByText('Deployment Type')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Database Username')).toBeInTheDocument();
  });

  it('updates dbUser', () => {
    const { onOptionsChange } = setup();

    fireEvent.change(screen.getByPlaceholderText('Database Username'), {
      target: { value: 'scott' },
    });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        jsonData: expect.objectContaining({ dbUser: 'scott' }),
      })
    );
  });

  it('updates password in secureJsonData', () => {
    const { onOptionsChange } = setup();

    fireEvent.change(screen.getByPlaceholderText('Database Password'), {
      target: { value: 'tiger' },
    });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        secureJsonData: { dbPassword: 'tiger' },
      })
    );
  });

  it('resets password and clears fields', () => {
    const { onOptionsChange } = setup({
      secureJsonFields: { dbPassword: true },
    });

    fireEvent.click(screen.getByText('Reset'));

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        secureJsonFields: expect.objectContaining({
          dbPassword: false,
        }),
        secureJsonData: expect.objectContaining({
          dbPassword: '',
          dbUser: '',
          dbConnectString: '',
        }),
      })
    );
  });

  it('renders BASIC auth fields', () => {
    setup({ jsonData: { queryAuth: 'BASIC' } });

    expect(screen.getByPlaceholderText('Database Hostname')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Database Port')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Database Service Name')).toBeInTheDocument();
  });

  it('updates BASIC auth fields', () => {
    const { onOptionsChange } = setup({ jsonData: { queryAuth: 'BASIC' } });

    fireEvent.change(screen.getByPlaceholderText('Database Hostname'), {
      target: { value: 'localhost' },
    });

    fireEvent.change(screen.getByPlaceholderText('Database Port'), {
      target: { value: '1521' },
    });

    fireEvent.change(screen.getByPlaceholderText('Database Service Name'), {
      target: { value: 'ORCL' },
    });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        jsonData: expect.objectContaining({
          dbHostName: 'localhost',
        }),
      })
    );
  });

  it('updates deployment type', () => {
    const { onOptionsChange } = setup();

    const selects = screen.getAllByTestId('mock-select');

    // second select = Deployment Type
    fireEvent.change(selects[1], {
      target: { value: 'ADB' },
    });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        jsonData: expect.objectContaining({
          deploymentType: 'ADB',
        }),
      })
    );
  });

  it('updates queryAuth when auth select changes', () => {
    const { onOptionsChange } = setup();

    const selects = screen.getAllByTestId('mock-select');

    fireEvent.change(selects[0], {
      target: { value: 'BASIC' },
    });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        jsonData: expect.objectContaining({
          queryAuth: 'BASIC',
        }),
      })
    );
  });

  it('updates deploymentType when deployment select changes', () => {
    const { onOptionsChange } = setup();

    const selects = screen.getAllByTestId('mock-select');

    fireEvent.change(selects[1], {
      target: { value: 'ADB' },
    });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        jsonData: expect.objectContaining({
          deploymentType: 'ADB',
        }),
      })
    );
  });

  it('resets secure fields when password is reset', () => {
    const { onOptionsChange } = setup({
      secureJsonFields: { dbPassword: true },
      secureJsonData: { dbPassword: 'secret' },
    });

    fireEvent.click(screen.getByRole('button'));

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({
        secureJsonFields: expect.objectContaining({
          dbPassword: false,
        }),
        secureJsonData: expect.objectContaining({
          dbPassword: '',
        }),
      })
    );
  });
});
