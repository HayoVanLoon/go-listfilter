# List Filter Parser

*Experimental, may change in backwards-incompatible ways*

A library for parsing filter expressions as found in list and search web calls.

Examples of filter strings:

```text
  "foo=bar"
  "foo.bar=bla"
  "foo=bar AND bla=vla"
  "foo>bar AND foo=bar"
```

The filter string should adher to the following grammar:

```text
Filter:
      <nil>
      Conditions
  Conditions:
      Condition { Separator Conditions }
  Separator:
 	 Space 'AND' Space
  Condition:
      FullName Operator Value
  FullName:
      NameParts
  NameParts:
      Name
      Name NameSeparator NameParts
  NameSeparator:
      '.'
  Name:
      regex([a-zA-Z][a-zA-Z0-9_]*)
  Operator:
      regex([^a-zA-Z0-9_].*)
  Value
      NormalValue | QuotedValue
  NormalValue
      regex([^separator\s]*)
  QuotedValue
      '"' Escaped '"'
  Escaped
      <nil>
      NormalChar Escaped
      EscapedChar Escaped
  EscapedChar
      '\\'
      '\"'
  NormalChar
      <not eChar>
```

An empty string is considered a valid input and will result in an empty Filter.

# License

Copyright 2022 Hayo van Loon

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License. You may obtain a copy of the
License at

       http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.
