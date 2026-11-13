package freemagpie

import (
	"testing"

	"github.com/yetone/magpie/internal/testenv"
)

// The tests run in a home of their own: the cache and the states are
// files beside magpie's others, and no test of them touches a real one.
func TestMain(m *testing.M) { testenv.Main(m) }
