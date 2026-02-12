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

// This is the component that we see for writing promql/sql queries
// after selecting our plugin.

import React, { ChangeEvent, PureComponent } from 'react';
import { InlineFormLabel, TextArea, Select, LegacyForms } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from './datasource';
import { DataSourceOptionsObj, QueryObj, InData } from './types';

import { AutoCompleteContainer, Input, AutoCompleteItem, AutoCompleteItemButton } from './styles';

const { Switch } = LegacyForms;

const QUERYLANG_OPTIONS: Array<SelectableValue<string>> = [
  { label: 'PROMQL', value: 'promql' },
  { label: 'SQL', value: 'sql' },
];

type Props = QueryEditorProps<DataSource, QueryObj, DataSourceOptionsObj>;

interface QueryEditorState {
  search: {
    text: string;
    suggestions: InData[];
  };
  isComponentVisible: boolean;
  isExecChecked: boolean;
}

export class QueryEditor extends PureComponent<Props, QueryEditorState> {
  constructor(props: Props) {
    super(props);
    this.state = {
      search: {
        text: '',
        suggestions: [] as InData[],
      },
      isComponentVisible: true,
      isExecChecked: true,
    };
  }
  labelList: InData[] = [];

  componentDidMount() {
    this.startLabelPolling();
  }

  componentWillUnmount() {
    clearInterval(this.labelTimer);
  }

  labelTimer: any;

  startLabelPolling() {
    const load = async () => {
      const data = await this.props.datasource.fetchStaticLabels();
      this.labelList = data; // store in class variable
    };

    load(); // initial fetch
    this.labelTimer = setInterval(load, 60000); // poll every 60 sec
  }

  //this function handles the event when sql query is changed/written
  onSqlTextChanged = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const { onChange, query } = this.props;
    onChange({ ...query, exprSql: event.target.value });
    //onChange({ ...query, expr: event.target.value });
  };

  //this function fires the event in backend to execute the query on sql change.
  onSqlBlur = () => {
    const { onRunQuery } = this.props;
    onRunQuery();
  };

  //This function fetches the value of toggle which determines
  //whether to convert the sql results into timeseries format.
  //Basically it changes a flag called "convertSqlResults" which is
  //used in backend to determine whether to return tabular
  //results or dataframe results for the current sql qurey.
  onInstantChange = (e: React.SyntheticEvent<HTMLInputElement>) => {
    const instant = (e.target as HTMLInputElement).checked;
    const { onChange, query, onRunQuery } = this.props;
    //changes the value of convertSqlResults by fetching it from ui
    onChange({ ...query, convertSqlResults: instant });
    // executes the query
    onRunQuery();
  };

  //This function switches between sql and promql language. This
  //is the handler of dropdown that as per the selected value changes
  // a flag called "queryLang" which is used in backend while running
  //the query. Also depending on this we show sql input textarea or
  //promql input as per the selection of user
  onQueryLangChange = (option: SelectableValue<string>) => {
    const { onChange, query, onRunQuery } = this.props;
    //changes the value of queryLang by fetching it from UI
    onChange({ ...query, queryLang: option.value });
    // executes the query
    onRunQuery();
  };

  //This function sets the value of legendFormat.
  onLegendTextChangedProm = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const { onChange, query } = this.props;
    //changes the value of legendFormat by fetching it from UI
    onChange({ ...query, legendFormatProm: value });
  };

  onLegendTextChangedSql = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const { onChange, query } = this.props;
    //changes the value of legendFormat by fetching it from UI
    onChange({ ...query, legendFormatSql: value });
  };

  //This function fires the query when legendtext is changed.
  onLegendBlur = () => {
    const { onRunQuery } = this.props;
    onRunQuery();
  };

  //This function sets the value of state variable "isComponentVisible" based on
  //passed value
  setIsComponentVisible = (val: boolean) => {
    this.setState({ isComponentVisible: val });
  };

  //this function handles the even when promql query is changes/written
  //and fires the event in backend to execute the query. Also it handles
  //the suggestion part of promql query.
  onPromTextChanged = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    let suggestionsNew: InData[] = []; //creates an empty array for suggestions

    if (value.length > 0 && !value.includes('(') && !value.includes(')')) {
      const regex = new RegExp(`^${value}`, 'i');
      suggestionsNew = this.labelList.sort().filter((v: InData) => regex.test(v.name));
    }
    //the above if condition check if entered text is not empty and then
    //populate new suggestions list with the json data using regular expression
    //in such a way that prefix of input matches with suggestions list.
    this.setIsComponentVisible(true);
    //finally we set the new suggestions list in state to display
    this.setState((prevState: QueryEditorState) => ({
      search: {
        suggestions: suggestionsNew,
        text: value,
      },
    }));

    const { onChange, query } = this.props;
    //replaces the querytext with value of input box
    onChange({ ...query, exprProm: value });
    //onChange({ ...query, expr: value });
  };

  // This function fires the query when promql is changed.
  onPromTextBlur = () => {
    const { onRunQuery } = this.props;
    onRunQuery();
  };

  //This function sets the value of stepText.
  onStepTextChangedProm = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const { onChange, query } = this.props;
    //changes the value of stepText by fetching it from UI
    onChange({ ...query, stepTextProm: value });
  };

  //This function sets the value of stepText.
  onStepTextChangedSql = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const { onChange, query } = this.props;
    //changes the value of stepText by fetching it from UI
    onChange({ ...query, stepTextSql: value });
  };

  // This function fires the query when stepText is changed.
  onStepBlur = () => {
    const { onChange, query } = this.props;

    let vsql = Number(this.props.query.stepTextSql);
    if (isNaN(vsql) || vsql < 10) {
      onChange({ ...query, stepTextSql: '10' });
    }

    let vprom = Number(this.props.query.stepTextProm);
    if (isNaN(vprom) || vprom < 10) {
      onChange({ ...query, stepTextProm: '10' });
    }

    const { onRunQuery } = this.props;
    onRunQuery();
  };

  //This function sets the value of prefetchCountText
  onPrefetchCountTextChanged = (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const { onChange, query } = this.props;
    //changes the value of prefetchCountText by fetching it from UI
    onChange({ ...query, prefetchCountText: value });
  };

  // This function fires the query when prefetchCountText is changed.
  onPrefetchCountBlur = () => {
    const { onRunQuery } = this.props;
    onRunQuery();
  };

  //This method is called when user click any of the suggestion from list
  //of suggestions provided to him. After this the text area is replaced
  //with the selected suggestion and query is executed
  suggestionSelected = (value: InData) => {
    //hide the suggestion box as soon as any suggestion is selected
    this.setIsComponentVisible(false);
    //set the suggestion list in state as empty.
    this.setState((prevState: QueryEditorState) => ({
      search: {
        suggestions: [],
        text: value.name,
      },
    }));
    const { onChange, query, onRunQuery } = this.props;
    //change the querytex with selected suggestion.
    onChange({ ...query, exprProm: value.name });
    //executes the query.
    onRunQuery();
  };

  render() {
    var exprProm = this.props.query.exprProm;
    var exprSql = this.props.query.exprSql;
    var legendFormatProm = this.props.query.legendFormatProm;
    var legendFormatSql = this.props.query.legendFormatSql;
    var stepTextProm = this.props.query.stepTextProm;
    var stepTextSql = this.props.query.stepTextSql;
    var prefetchCountText = this.props.query.prefetchCountText;
    var checkedVal = this.props.query.convertSqlResults;

    var curLang = this.props.query.queryLang ?? 'promql';

    const { suggestions } = this.state.search;

    return (
      <div>
        <div className="gf-form">
          <div style={{ width: '115px' }}>
            {curLang === 'promql' ? (
              <Select
                className={'select-container'}
                isSearchable={false}
                options={QUERYLANG_OPTIONS}
                defaultValue={{ label: 'PROMQL', value: 'promql' }}
                onChange={this.onQueryLangChange}
              />
            ) : (
              <Select
                className={'select-container'}
                isSearchable={false}
                options={QUERYLANG_OPTIONS}
                defaultValue={{ label: 'SQL', value: 'sql' }}
                onChange={this.onQueryLangChange}
              />
            )}
          </div>

          <div style={{ marginLeft: '25px', width: 'calc(100% - 200px)' }}>
            {curLang === 'promql' ? (
              //If selected language is Promql display this input box
              <div>
                <div style={{ width: '115px', paddingBottom: '5px' }}>{}</div>
                <Input
                  id="input"
                  autoComplete="off"
                  value={exprProm || ''}
                  onChange={this.onPromTextChanged}
                  onBlur={this.onPromTextBlur}
                  className="gf-form-input"
                  placeholder="Enter Promql Query"
                  type="text"
                  style={{ paddingBottom: '5px' }}
                />
              </div>
            ) : (
              <div>
                <TextArea
                  //if selected language is SQL display this Textarea as sql
                  //queries are generally bigger
                  value={exprSql || ''}
                  placeholder="Enter Sql Query"
                  onChange={this.onSqlTextChanged}
                  onBlur={this.onSqlBlur}
                  label="Query Text"
                />
                <Switch //toggle to convert sql results to dataframe format
                  label="Time Series Mode"
                  checked={checkedVal === undefined ? true : checkedVal}
                  onChange={this.onInstantChange}
                />
              </div>
            )}
          </div>

          {/*The following code is for the list of suggestions*/}
          <div>
            <div
              onClick={() => this.setIsComponentVisible(false)}
              style={{
                display: this.state.isComponentVisible ? 'block' : 'none',
                width: '200vw',
                height: '200vh',
                backgroundColor: 'transparent',
                position: 'fixed',
                zIndex: 0,
                top: 0,
                left: 0,
              }}
            />
            {suggestions.length > 0 && this.state.isComponentVisible && (
              <AutoCompleteContainer>
                {suggestions.map((item: InData) => (
                  <AutoCompleteItem key={item.name}>
                    <AutoCompleteItemButton key={item.name} onClick={() => this.suggestionSelected(item)}>
                      {item.name}
                    </AutoCompleteItemButton>
                  </AutoCompleteItem>
                ))}
              </AutoCompleteContainer>
            )}
          </div>
        </div>

        {curLang === 'promql' ? (
          <div>
            <div className="gf-form">
              <InlineFormLabel width={7}>Legend</InlineFormLabel>
              <div style={{ marginLeft: '25px', width: '400px' }}>
                <Input
                  id="inputLegend"
                  value={legendFormatProm || ''}
                  onChange={this.onLegendTextChangedProm}
                  onBlur={this.onLegendBlur}
                  className="gf-form-input width-32"
                  placeholder="eg. {{objectName}}"
                  type="text"
                />
              </div>
            </div>
            <div className="gf-form">
              <InlineFormLabel width={7}>Step Size</InlineFormLabel>
              <div style={{ marginLeft: '25px', width: '400px' }}>
                <Input
                  id="stepSize"
                  value={stepTextProm || ''}
                  onChange={this.onStepTextChangedProm}
                  onBlur={this.onStepBlur}
                  className="gf-form-input width-32"
                  placeholder="Steps Size in Seconds, Min:10 Default:10"
                  type="number"
                  min="1"
                  max="100000"
                  step="1"
                />
              </div>
            </div>
          </div>
        ) : null}

        {curLang === 'sql' ? (
          <div>
            <div className="gf-form">
              <InlineFormLabel width={7}>Legend</InlineFormLabel>
              <div style={{ marginLeft: '25px', width: '400px' }}>
                <Input
                  id="inputLegend"
                  value={legendFormatSql || ''}
                  onChange={this.onLegendTextChangedSql}
                  onBlur={this.onLegendBlur}
                  className="gf-form-input width-32"
                  placeholder="eg. {{objectName}}"
                  type="text"
                />
              </div>
            </div>
            <div className="gf-form">
              <InlineFormLabel width={7}>Step Size</InlineFormLabel>
              <div style={{ marginLeft: '25px', width: '400px' }}>
                <Input
                  id="stepSize"
                  value={stepTextSql || ''}
                  onChange={this.onStepTextChangedSql}
                  onBlur={this.onStepBlur}
                  className="gf-form-input width-32"
                  placeholder="Steps Size in Seconds, Min:10 Default:10"
                  type="number"
                  min="10"
                  max="100000"
                  step="1"
                />
              </div>
            </div>
            <div className="gf-form">
              <InlineFormLabel width={7}>Prefetch Count</InlineFormLabel>
              <div style={{ marginLeft: '25px', width: '400px' }}>
                <Input
                  id="prefetchCount"
                  value={prefetchCountText || ''}
                  onChange={this.onPrefetchCountTextChanged}
                  onBlur={this.onPrefetchCountBlur}
                  className="gf-form-input width-32"
                  placeholder="SQL Prefect Count Default:100"
                  type="number"
                  min="1"
                  max="100000"
                  step="1"
                />
              </div>
            </div>
          </div>
        ) : null}
      </div>
    );
  }
}
