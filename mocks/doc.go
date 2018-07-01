/*
Package mocks will have all the mocks of the library, we'll try to use mocking using blackbox
testing and integration tests whenever is possible.
*/
package mocks // import "github.com/slok/kutator/mocks"

// mutator mocks.
//go:generate mockery -output ./mutate -outpkg mutate -dir ../pkg/mutate -name Mutator
