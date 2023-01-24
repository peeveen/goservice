# Go Service

Helper library for writing services in Go.

## Usage

Call `RunService` or `RunServiceFunction`, passing in:

- a service config (what it's called, how it's displayed)
- an object or function containing the service functionality
- a logging config (optional, can be `nil`)

The object containing the service functionality must adhere to the `ServiceRunner` interface.

If you pass a function, it will be wrapped in the `Controller` helper object.
