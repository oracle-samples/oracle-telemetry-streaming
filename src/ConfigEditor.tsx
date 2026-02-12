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

import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms, InlineField, InlineFormLabel, Select } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps, SelectableValue } from '@grafana/data';
import { DataSourceOptionsObj, SecureJsonData } from './types';

const { SecretFormField, FormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<DataSourceOptionsObj> {}

interface State {}

const AUTH_OPTIONS: Array<SelectableValue<string>> = [
  { label: 'TNS', value: 'TNS' },
  { label: 'BASIC', value: 'BASIC' },
];

const DEPLOYMENT_OPTIONS: Array<SelectableValue<string>> = [
  { label: 'ON-PREM', value: 'ON-PREM' },
  { label: 'ADB', value: 'ADB' },
];

const getSelectValue = (options: Array<SelectableValue<string>>, value: string) =>
  options.find((option) => option.value === value) || options[0];

export class ConfigEditor extends PureComponent<Props, State> {
  onDBUserChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      dbUser: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onDBConnectionIdentifierChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      dbConnectString: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onDBHostNameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      dbHostName: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onDBPortNameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      dbPortName: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onDBServiceNameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      dbServiceName: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onAuthChange = (option: SelectableValue<string>) => {
    const { onOptionsChange, options } = this.props;
    //changes the value of queryAuth by fetching it from UI
    const jsonData = {
      ...options.jsonData,
      queryAuth: option.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onDeploymentChange = (option: SelectableValue<string>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      deploymentType: option.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  // Secure field (only sent to the backend)
  onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: {
        dbPassword: event.target.value,
      },
    });
  };

  onResetPassword = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        dbPassword: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        dbPassword: '',
        dbUser: '',
        dbConnectString: '',
      },
    });
  };

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as SecureJsonData;
    const curAuth = options.jsonData.queryAuth;
    const currentAuthValue = curAuth || 'TNS';
    const deployment = options.jsonData.deploymentType || 'ON-PREM';
    const authSelectValue = getSelectValue(AUTH_OPTIONS, currentAuthValue);
    const deploymentSelectValue = getSelectValue(DEPLOYMENT_OPTIONS, deployment);

    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <InlineFormLabel width={10} tooltip="Choose between Basic or TNS">
            Connection Type
          </InlineFormLabel>
          <InlineField>
            <Select
              className={'select-container'}
              isSearchable={false}
              options={AUTH_OPTIONS}
              value={authSelectValue}
              onChange={this.onAuthChange}
              width={24}
            />
          </InlineField>
        </div>
        <div className="gf-form">
          <InlineFormLabel width={10} tooltip="Choose where your database is hosted">
            Deployment Type
          </InlineFormLabel>
          <InlineField>
            <Select
              className={'select-container'}
              isSearchable={false}
              options={DEPLOYMENT_OPTIONS}
              value={deploymentSelectValue}
              onChange={this.onDeploymentChange}
              width={24}
            />
          </InlineField>
        </div>

        <div>
          <div className="gf-form">
            <FormField
              label="Username"
              labelWidth={10}
              inputWidth={20}
              onChange={this.onDBUserChange}
              value={jsonData.dbUser || ''}
              placeholder="Database Username"
            />
          </div>

          <div className="gf-form-inline">
            <div className="gf-form">
              <SecretFormField
                isConfigured={(secureJsonFields && secureJsonFields.dbPassword) as boolean}
                value={secureJsonData.dbPassword || ''}
                label="Password"
                placeholder="Database Password"
                labelWidth={10}
                inputWidth={20}
                onReset={this.onResetPassword}
                onChange={this.onPasswordChange}
              />
            </div>
          </div>

          {currentAuthValue === 'TNS' ? (
            <div className="gf-form">
              <FormField
                label="Connection Identifier"
                labelWidth={10}
                inputWidth={20}
                onChange={this.onDBConnectionIdentifierChange}
                value={jsonData.dbConnectString || ''}
                placeholder="Database Connection Identifier"
              />
            </div>
          ) : (
            <>
              <div className="gf-form">
                <FormField
                  label="Hostname"
                  labelWidth={10}
                  inputWidth={20}
                  onChange={this.onDBHostNameChange}
                  value={jsonData.dbHostName || ''}
                  placeholder="Database Hostname"
                />
              </div>
              <div className="gf-form">
                <FormField
                  label="Port"
                  labelWidth={10}
                  inputWidth={12}
                  onChange={this.onDBPortNameChange}
                  value={jsonData.dbPortName || ''}
                  placeholder="Database Port"
                />
              </div>
              <div className="gf-form">
                <FormField
                  label="Service Name"
                  labelWidth={10}
                  inputWidth={20}
                  onChange={this.onDBServiceNameChange}
                  value={jsonData.dbServiceName || ''}
                  placeholder="Database Service Name"
                />
              </div>
            </>
          )}
        </div>
      </div>
    );
  }
}
