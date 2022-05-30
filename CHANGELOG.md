# v0.3.0

## Breaking Changes

* `Filter` is now an interface type (with some map-like accessors)
* Renamed `FilterParser` to just `Parser`

## New Functionality

* Added support for OR between conditions
* The `Filter.First` method will return the first condition encountered. From
  there, you can use the `And` or `Or` methods to move to the next conditions.
  Outside this pattern, these methods will have little semantic value.

# v0.2.0

## Breaking Changes

* Changed separator from `,` to ` AND ` to be in line
  with [source behaviour](https://cloud.google.com/service-infrastructure/docs/service-consumer-management/reference/rest/v1/services/search#query-parameters
  )

## New Functionality

* Parser option for converting field names to snake_case
* Parser option for converting field names to camelCase

# v0.1.0

* First version
