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



import styled, { css } from "styled-components";

export const Root = styled.div`
  position: relative;
  width: 320px;
`;

export const baseButtonMixin = css`
  background: none;
  border: none;
  padding: 0;
`;

export const ValueWrapper = styled.input`
  width: 100%;
  padding-left: 8px;
  padding-right: 32px;
  height: 32px;
  box-sizing: border-box;
  border-radius: 1px;
  border: 1px solid #b6c1ce;
  line-height: 32px;
`;

export const AutoCompleteIcon = styled.span`
  position: absolute;
  top: 0;
  right: 0;
  height: 32px;
  width: 32px;
  transition: all 150ms linear;
  transform: ${(props: any) => (props.isOpen ? "rotate(0.5turn)" : "none")};
  transform-origin: center;
  display: flex;

  svg {
    margin: auto;
  }

  ${ValueWrapper}:focus + & {
    color: ${(props: any) => props.color || "0063cc"};
    fill: ${(props: any) => props.fill || "0063cc"};
  }
`;

export const AutoCompleteContainer = styled.ul`
  background: #181b1f;
  padding: 8px 0;
  list-style-type: none;
  min-width: 400px;
  position: absolute;
  top: 100%;
  left: 172px;
  border: 1px solid #97a0ab;
  border-radius: 10px;
  margin: 2px;
  box-sizing: border-box;
  max-height: 280px;
  overflow-y: auto;
  z-index: 1;
`;

export const AutoCompleteItem = styled.li`
  padding: 0 24px;
  width: 100%;
  box-sizing: border-box;
  &:hover {
    background-color: #ebf4ff;
  }
`;

export const AutoCompleteItemButton = styled.button`
  ${baseButtonMixin} width: 100%;
  line-height: 32px;
  text-align: left;
  &:active {
    outline: none;
    color: #0076f5;
  }
`;
export const Input = styled(ValueWrapper)`
  transition: border-color 150ms linear;

  &:focus {
    border-color: #0063cc;
    outline: none;

    + ${AutoCompleteIcon} {
      color: ${(props: any) => props.color || "0063cc"};
      fill: ${(props: any) => props.fill || "0063cc"};
    }
  }
`;
