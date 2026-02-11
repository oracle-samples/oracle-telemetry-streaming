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



//This is the component that gets rendered when we create a query variable
//It is the input box of query editor with its functionalities
import { SelectableValue } from "@grafana/data";

import React, { useState } from "react";
import { VariableQueryObject } from "./types";
import { Select, InlineFormLabel } from "@grafana/ui";
interface VariableQueryProps {
  query: VariableQueryObject;
  onChange: (query: VariableQueryObject, definition: string) => void;
}

export const VariableQueryEditor: React.FC<VariableQueryProps> = ({
  onChange,
  query,
}) => {
  const [state, setState] = useState(query);
  const QUERY_OPTIONS: Array<SelectableValue<string>> = [
    { label: "PROMQL", value: "promql" },
  ];

  //this saves the query and  calls the backend fetch tags
  const saveQuery = () => {
    onChange(state, `${state.query}`);
  };

  const handleChangeLang = (option: SelectableValue<string>) => {
    console.log("here");
  };

  //this function handles the change in input of query text box and saves it in
  //state
  const handleChange = (event: React.FormEvent<HTMLInputElement>) => {
    setState({
      ...state,
      [event.currentTarget.name]: event.currentTarget.value,
    });
  };

  return (
    <>
      {/* Row 1: Query Language */}
      <div className="gf-form">
        <InlineFormLabel width={10} tooltip="Query Language To fetch Variables">
          Query Language
        </InlineFormLabel>
        <Select
          className="select-container"
          isSearchable={false}
          defaultValue={{ label: "PROMQL", value: "promql" }}
          onChange={handleChangeLang}
          options={QUERY_OPTIONS}
          width={35}
        />
      </div>
      {/* Row 2: Query */}
      <div className="gf-form">
        <InlineFormLabel width={10}>Query</InlineFormLabel>
        <input
          name="query"
          className="gf-form-input"
          onBlur={saveQuery}
          onChange={handleChange}
          value={state.query}
        />
      </div>
    </>
  );
};
