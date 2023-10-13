package subnet

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextAvailableSubnet(t *testing.T) {
	assert := assert.New(t)
	calc := NewCalculator()
	calc.AddPool(netip.MustParsePrefix("fd18:fad4:bce5:4400::/56"))
	next, err := calc.NextAvailableIPv6Subnet(64)
	if assert.NoError(err) {
		assert.Equal("fd18:fad4:bce5:4400::/64", next.String())
	}
	next, err = calc.NextAvailableIPv6Subnet(64)
	if assert.NoError(err) {
		assert.Equal("fd18:fad4:bce5:4401::/64", next.String())
	}
	next, err = calc.NextAvailableIPv6Subnet(64)
	if assert.NoError(err) {
		assert.Equal("fd18:fad4:bce5:4402::/64", next.String())
	}
}

func TestNextAvailableSubnetWithAllocated(t *testing.T) {
	assert := assert.New(t)
	calc := NewCalculator()
	calc.AddPool(netip.MustParsePrefix("fd18:fad4:bce5:4400::/56"))
	calc.AddAllocatedPrefix(netip.MustParsePrefix("fd18:fad4:bce5:4400::/64"))
	next, err := calc.NextAvailableIPv6Subnet(64)
	if assert.NoError(err) {
		assert.Equal("fd18:fad4:bce5:4401::/64", next.String())
	}
	next, err = calc.NextAvailableIPv6Subnet(64)
	if assert.NoError(err) {
		assert.Equal("fd18:fad4:bce5:4402::/64", next.String())
	}
	next, err = calc.NextAvailableIPv6Subnet(64)
	if assert.NoError(err) {
		assert.Equal("fd18:fad4:bce5:4403::/64", next.String())
	}
	next, err = calc.NextAvailableIPv6Subnet(64)
	if assert.NoError(err) {
		assert.Equal("fd18:fad4:bce5:4404::/64", next.String())
	}
}
