package slippage

import "github.com/quagmt/udecimal"

// Default slippage rates. It works on a percentage basis
// 100 means unaffected, 95 would mean 95%
var (
	DefaultMaximumSlippagePercent = udecimal.MustFromInt64(100, 0)
	DefaultMinimumSlippagePercent = udecimal.MustFromInt64(100, 0)
)
