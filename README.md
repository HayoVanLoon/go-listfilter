# List Filter Parser

*Experimental, may change in backwards-incompatible ways*

A library for parsing filter expressions as found in list and search web calls.

The string should adher to the following grammar:
```text
Filter ->           <nil> | conditions
conditions ->       Condition | Condition separator Conditions
separator ->        ,
Condition ->        fullName operator value
fullName ->         nameParts
nameParts ->        name | name nameSeparator nameParts
nameSeparator ->    .
name ->             regex([a-zA-Z][a-zA-Z0-9_]*)
operator ->         regex([^a-zA-Z0-9_].*)
value ->            normalValue | quotedValue
normalValue ->      regex([^separator]*)
quotedValue ->      " escaped "
escaped ->          <nil> | nChar escaped | eChar escaped
eChar ->            \\ | \"
nChar ->            <not eChar>
```

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
