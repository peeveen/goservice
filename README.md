# Go Service

Helper library for writing services in Go.

## Usage

Call `RunService` or `RunServiceFunction`, passing in:

- a service config (what it's called, how it's displayed)
- a function containing or providing the service functionality
- a logging config (optional, can be `nil`)

The object containing the service functionality must adhere to the `ServiceRunner` interface.
