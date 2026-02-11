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



import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryEditor } from "../QueryEditor";
import { QueryObj } from "../types";
/* ---------- mocks ---------- */
const mockDatasource = {
  fetchStaticLabels: jest
    .fn()
    .mockResolvedValue([
      { name: "cpu_usage" },
      { name: "cpu_idle" },
      { name: "memory_free" },
    ]),
};
/* ---------- helpers ---------- */
const baseQuery: QueryObj = {
  queryLang: "promql",
  exprProm: "",
  exprSql: "",
  legendFormatProm: "",
  legendFormatSql: "",
  stepTextProm: "10",
  stepTextSql: "10",
  prefetchCountText: "100",
  convertSqlResults: true,
};
function setup(queryOverrides?: Partial<QueryObj>) {
  const onChange = jest.fn();
  const onRunQuery = jest.fn();
  const query = { ...baseQuery, ...queryOverrides };
  render(
    <QueryEditor
      query={query}
      datasource={mockDatasource as any}
      onChange={onChange}
      onRunQuery={onRunQuery}
    />
  );
  return { onChange, onRunQuery };
}
/* ---------- tests ---------- */
describe("QueryEditor", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });
  /* ===== Rendering ===== */
  it("renders PROMQL editor by default", () => {
    setup();
    expect(
      screen.getByPlaceholderText("Enter Promql Query")
    ).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("eg. {{objectName}}")
    ).toBeInTheDocument();
  });
  it("renders SQL editor when queryLang is sql", () => {
    setup({ queryLang: "sql" });
    expect(screen.getByPlaceholderText("Enter Sql Query")).toBeInTheDocument();
    expect(screen.getByText("Time Series Mode")).toBeInTheDocument();
  });
  /* ===== PROMQL behavior ===== */
  it("updates promql expression when typing", () => {
    const { onChange } = setup();
    fireEvent.change(screen.getByPlaceholderText("Enter Promql Query"), {
      target: { value: "cpu" },
    });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ exprProm: "cpu" })
    );
  });
  it("runs query when promql input loses focus", () => {
    const { onRunQuery } = setup();
    fireEvent.blur(screen.getByPlaceholderText("Enter Promql Query"));
    expect(onRunQuery).toHaveBeenCalled();
  });
  it("updates promql legend", () => {
    const { onChange } = setup();
    fireEvent.change(screen.getByPlaceholderText("eg. {{objectName}}"), {
      target: { value: "{{host}}" },
    });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ legendFormatProm: "{{host}}" })
    );
  });
  it("enforces minimum step size on blur (promql)", () => {
    const { onChange, onRunQuery } = setup({ stepTextProm: "2" });
    fireEvent.blur(
      screen.getByPlaceholderText("Steps Size in Seconds, Min:10 Default:10")
    );
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ stepTextProm: "10" })
    );
    expect(onRunQuery).toHaveBeenCalled();
  });
  /* ===== SQL behavior ===== */
  it("updates sql text when typing", () => {
    const { onChange } = setup({ queryLang: "sql" });
    fireEvent.change(screen.getByPlaceholderText("Enter Sql Query"), {
      target: { value: "select * from dual" },
    });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ exprSql: "select * from dual" })
    );
  });
  it("toggles Time Series Mode and runs query", () => {
    const { onChange, onRunQuery } = setup({
      queryLang: "sql",
      convertSqlResults: false,
    });
    fireEvent.click(screen.getByLabelText("Time Series Mode"));
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ convertSqlResults: true })
    );
    expect(onRunQuery).toHaveBeenCalled();
  });
  it("updates SQL prefetch count", () => {
    const { onChange, onRunQuery } = setup({ queryLang: "sql" });
    fireEvent.change(
      screen.getByPlaceholderText("SQL Prefect Count Default:100"),
      { target: { value: "200" } }
    );
    fireEvent.blur(
      screen.getByPlaceholderText("SQL Prefect Count Default:100")
    );
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ prefetchCountText: "200" })
    );
    expect(onRunQuery).toHaveBeenCalled();
  });
  /* ===== Autocomplete ===== */
  it("fetches label suggestions when typing promql", async () => {
    const { onChange } = setup();
    const input = screen.getByPlaceholderText("Enter Promql Query");
    fireEvent.change(input, { target: { value: "cpu" } });
    await waitFor(() => {
      expect(mockDatasource.fetchStaticLabels).toHaveBeenCalled();
    });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ exprProm: "cpu" })
    );
  });

  it("handles empty label suggestions gracefully", async () => {
    mockDatasource.fetchStaticLabels.mockResolvedValueOnce([]);

    setup();

    fireEvent.change(screen.getByPlaceholderText("Enter Promql Query"), {
      target: { value: "cpu" },
    });

    await waitFor(() => {
      expect(mockDatasource.fetchStaticLabels).toHaveBeenCalled();
    });
  });

  it("keeps step size when value is >= min", () => {
    const { onChange } = setup({ stepTextProm: "20" });

    fireEvent.blur(
      screen.getByPlaceholderText("Steps Size in Seconds, Min:10 Default:10")
    );

    expect(onChange).not.toHaveBeenCalledWith(
      expect.objectContaining({ stepTextProm: "10" })
    );
  });
});
