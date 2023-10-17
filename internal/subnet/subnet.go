package subnet

import (
	"fmt"
	iradix "github.com/hashicorp/go-immutable-radix"
	"net/netip"
)

// Calculator stores radix trees of supernets and subnets.
type Calculator struct {
	IPv4Pools             *iradix.Tree
	AllocatedIPv4Prefixes *iradix.Tree
	IPv6Pools             *iradix.Tree
	AllocatedIPv6Prefixes *iradix.Tree
}

// NewCalculator creates a new Calculator from a list of supernets and subnets.
func NewCalculator() *Calculator {
	return &Calculator{
		IPv4Pools:             iradix.New(),
		AllocatedIPv4Prefixes: iradix.New(),
		IPv6Pools:             iradix.New(),
		AllocatedIPv6Prefixes: iradix.New(),
	}
}

func (c *Calculator) AddPool(prefix netip.Prefix) {
	addr := prefix.Addr().As16()
	bytes := make([]byte, len(addr))
	copy(bytes, addr[:])
	if prefix.Addr().Is4() {
		c.IPv4Pools, _, _ = c.IPv4Pools.Insert(bytes, prefix)
	} else {
		c.IPv6Pools, _, _ = c.IPv6Pools.Insert(bytes, prefix)
	}
}

func (c *Calculator) DeletePool(prefix netip.Prefix) {
	addr := prefix.Addr().As16()
	bytes := make([]byte, len(addr))
	copy(bytes, addr[:])
	if prefix.Addr().Is4() {
		c.IPv4Pools, _, _ = c.IPv4Pools.Delete(bytes)
	} else {
		c.IPv6Pools, _, _ = c.IPv6Pools.Delete(bytes)
	}
}

func (c *Calculator) AddAllocatedPrefix(prefix netip.Prefix) {
	addr := prefix.Addr().As16()
	bytes := make([]byte, len(addr))
	copy(bytes, addr[:])
	if prefix.Addr().Is4() {
		c.AllocatedIPv4Prefixes, _, _ = c.AllocatedIPv4Prefixes.Insert(bytes, prefix)
	} else {
		c.AllocatedIPv6Prefixes, _, _ = c.AllocatedIPv6Prefixes.Insert(bytes, prefix)
	}
}

func (c *Calculator) DeleteAllocatedPrefix(prefix netip.Prefix) {
	addr := prefix.Addr().As16()
	bytes := make([]byte, len(addr))
	copy(bytes, addr[:])
	if prefix.Addr().Is4() {
		c.AllocatedIPv4Prefixes, _, _ = c.AllocatedIPv4Prefixes.Delete(bytes)
	} else {
		c.AllocatedIPv6Prefixes, _, _ = c.AllocatedIPv6Prefixes.Delete(bytes)
	}
}

// PrefixInPools tests to see if a prefix is a part of any
// pools that have been added to the calculator.
func (c *Calculator) PrefixInPools(prefix netip.Prefix) bool {
	pool := c.IPv4Pools
	if prefix.Addr().Is6() {
		pool = c.IPv6Pools
	}
	result := false
	pool.Root().Walk(func(k []byte, v interface{}) bool {
		n, ok := v.(netip.Prefix)
		if !ok {
			panic("unexpected node type found in radix tree")
		}
		if n.Contains(prefix.Addr()) {
			result = true
			return true
		}
		return false
	})
	return result
}

// NextAvailableIPv4Subnet finds the first available IPv4 subnet of a given mask length
// from a list of subnets and supernets, and fails if none are available.
func (c *Calculator) NextAvailableIPv4Subnet(numBits int) (netip.Prefix, error) {
	// For each eligible subnet, walk the tree and determine if the subnet is
	// available for use, and return the first subnet that is available.
	sf := newSubnetV4Factory(c, numBits)
	defer sf.stop()

	for subnet := range sf.subnetsChan {
		if c.prefixAvailable(subnet) {
			addr := subnet.Addr().As16()
			bytes := make([]byte, len(addr))
			copy(bytes, addr[:])
			c.AllocatedIPv4Prefixes, _, _ = c.AllocatedIPv4Prefixes.Insert(bytes, subnet)
			return subnet, nil
		}
	}

	return netip.Prefix{}, fmt.Errorf("No eligible subnet with mask /%v found", numBits)
}

// NextAvailableIPv6Subnet finds the first available IPv6 subnet of a given mask length
// from a list of subnets and supernets, and fails if none are available.
func (c *Calculator) NextAvailableIPv6Subnet(numBits int) (netip.Prefix, error) {
	// For each eligible subnet, walk the tree and determine if the subnet is
	// available for use, and return the first subnet that is available.
	sf := newSubnetV6Factory(c, numBits)
	defer sf.stop()

	for subnet := range sf.subnetsChan {
		if c.prefixAvailable(subnet) {
			addr := subnet.Addr().As16()
			bytes := make([]byte, len(addr))
			copy(bytes, addr[:])
			c.AllocatedIPv6Prefixes, _, _ = c.AllocatedIPv6Prefixes.Insert(bytes, subnet)
			return subnet, nil
		}
	}

	return netip.Prefix{}, fmt.Errorf("No eligible subnet with mask /%v found", numBits)
}

// subnetAvailable tests to see if an IPNet is available in an existing tree of subnets.
func (c *Calculator) prefixAvailable(prefix netip.Prefix) bool {
	allocated := c.AllocatedIPv4Prefixes
	if prefix.Addr().Is6() {
		allocated = c.AllocatedIPv6Prefixes
	}
	result := true
	allocated.Root().Walk(func(k []byte, v interface{}) bool {
		n, ok := v.(netip.Prefix)
		if !ok {
			panic("unexpected node type found in radix tree")
		}
		// ones, _ := n.Mask.Size()
		// // fmt.Printf("net: %s   ones: %d   bits: %d\n", currentIPNet.String(), ones, bits)
		// if ones <= 16 {
		// 	return false
		// }
		if n.Contains(prefix.Addr()) {
			result = false
			return true
		}
		if prefix.Contains(n.Addr()) {
			result = false
			return true
		}
		return false
	})
	return result
}

type subnetFactory struct {
	supernets    *iradix.Tree
	prefixLength int
	subnetsChan  chan netip.Prefix
	doneChan     chan struct{}
}

func newSubnetV4Factory(c *Calculator, prefixLength int) *subnetFactory {
	sf := &subnetFactory{
		supernets:    c.IPv4Pools,
		prefixLength: prefixLength,
		subnetsChan:  make(chan netip.Prefix),
		doneChan:     make(chan struct{}),
	}
	go sf.run4()
	return sf
}

func newSubnetV6Factory(c *Calculator, prefixLength int) *subnetFactory {
	sf := &subnetFactory{
		supernets:    c.IPv6Pools,
		prefixLength: prefixLength,
		subnetsChan:  make(chan netip.Prefix),
		doneChan:     make(chan struct{}),
	}
	go sf.run6()
	return sf
}

func (sf *subnetFactory) stop() {
	close(sf.doneChan)
}

func (sf *subnetFactory) run4() {
	sf.supernets.Root().Walk(func(k []byte, v interface{}) bool {
		select {
		case <-sf.doneChan:
			return true
		default:
			n, ok := v.(netip.Prefix)
			if !ok {
				panic("unexpected node type found in radix tree")
			}
			addr := n.Addr().As4()
			newPrefix := netip.PrefixFrom(netip.AddrFrom4(addr), sf.prefixLength)
			sf.subnetsChan <- newPrefix
			for {
				addr = increment4(addr, sf.prefixLength)
				newPrefix = netip.PrefixFrom(netip.AddrFrom4(addr), sf.prefixLength)
				if !n.Contains(newPrefix.Addr()) {
					break
				}
				sf.subnetsChan <- newPrefix
			}
			return false
		}
	})
	close(sf.subnetsChan)
}

func (sf *subnetFactory) run6() {
	sf.supernets.Root().Walk(func(k []byte, v interface{}) bool {
		select {
		case <-sf.doneChan:
			return true
		default:
			n, ok := v.(netip.Prefix)
			if !ok {
				panic("unexpected node type found in radix tree")
			}
			addr := n.Addr().As16()
			newPrefix := netip.PrefixFrom(netip.AddrFrom16(addr), sf.prefixLength)
			sf.subnetsChan <- newPrefix
			for {
				addr = increment16(addr, sf.prefixLength)
				newPrefix = netip.PrefixFrom(netip.AddrFrom16(addr), sf.prefixLength)
				if !n.Contains(newPrefix.Addr()) {
					break
				}
				sf.subnetsChan <- newPrefix
			}
			return false
		}
	})
	close(sf.subnetsChan)
}

func increment4(a [4]byte, bit int) [4]byte {
	octet := (bit - 1) / 8
	val := uint16(128) >> ((bit - 1) - (octet * 8))
	sum16 := uint16(a[octet]) + val
	a[octet] = byte(sum16)
	carry := sum16 >> 8
	for {
		if carry == 0 {
			return a
		}
		octet--
		if octet < 0 {
			// overflow
			return [4]byte{}
		}
		sum16 = uint16(a[octet]) + carry
		a[octet] = byte(sum16)
		carry = sum16 >> 8
	}
}

func increment16(a [16]byte, bit int) [16]byte {
	octet := (bit - 1) / 8
	val := uint16(128) >> ((bit - 1) - (octet * 8))
	sum16 := uint16(a[octet]) + val
	a[octet] = byte(sum16)
	carry := sum16 >> 8
	for {
		if carry == 0 {
			return a
		}
		octet--
		if octet < 0 {
			// overflow
			return [16]byte{}
		}
		sum16 = uint16(a[octet]) + carry
		a[octet] = byte(sum16)
		carry = sum16 >> 8
	}
}
