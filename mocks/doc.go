/*
Package mocks will have all the mocks of the library, we'll try to use mocking using blackbox
testing and integration tests whenever is possible.
*/
package mocks // import "github.com/slok/kubewebhook/mocks"

// Mutating mocks.
//go:generate mockery -output ./webhook/mutating -outpkg mutating -dir ../pkg/webhook/mutating -name Mutator

// Validating mocks.
//go:generate mockery -output ./webhook/validating -outpkg validating -dir ../pkg/webhook/validating -name Validator

// Webhook mocks.
//go:generate mockery -output ./webhook -outpkg webhook -dir ../pkg/webhook -name Webhook

// Observability mocks.
//go:generate mockery -output ./observability/metrics -outpkg metrics -dir ../pkg/observability/metrics -name Recorder
