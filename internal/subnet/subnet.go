package subnet

import (
	"errors"
	"fmt"
	"net/netip"

	iradix "github.com/hashicorp/go-immutable-radix"
)

// Calculator stores radix trees of supernets and subnets.
type Calculator struct {
	Pools             *iradix.Tree
	AllocatedPrefixes *iradix.Tree
	Mode              Mode
}

type Mode int

const (
	ModeUnknown Mode = iota
	ModeV4
	ModeV6
)

// New creates a new Calculator from a list of supernets and subnets.
func NewCalculator() *Calculator {
	return &Calculator{
		Pools:             iradix.New(),
		AllocatedPrefixes: iradix.New(),
		Mode:              ModeUnknown, // Mode will be discovered by the first prefix added.
	}
}

func (c *Calculator) checkMode(prefix netip.Prefix) error {
	switch {
	case c.Mode == ModeUnknown && prefix.Addr().Is4():
		c.Mode = ModeV4
	case c.Mode == ModeUnknown && prefix.Addr().Is6():
		c.Mode = ModeV6
	case c.Mode == ModeV4 && !prefix.Addr().Is4():
		return errors.New("Attempting to add an IPv6 CIDR to an IPv4 calculator, all CIDRs must be of the same type.")
	case c.Mode == ModeV6 && !prefix.Addr().Is6():
		return errors.New("Attempting to add an IPv4 CIDR to an IPv6 calculator, all CIDRs must be of the same type.")
	}
	return nil
}

func (c *Calculator) AddPool(prefix netip.Prefix) error {
	if err := c.checkMode(prefix); err != nil {
		return err
	}
	addr := prefix.Addr().As16()
	bytes := make([]byte, len(addr))
	copy(bytes, addr[:])
	c.Pools, _, _ = c.Pools.Insert(bytes, prefix)
	return nil
}

func (c *Calculator) DeletePool(prefix netip.Prefix) {
	addr := prefix.Addr().As16()
	bytes := make([]byte, len(addr))
	copy(bytes, addr[:])
	c.Pools, _, _ = c.Pools.Delete(bytes)
}

func (c *Calculator) AddAllocatedPrefix(prefix netip.Prefix) error {
	if err := c.checkMode(prefix); err != nil {
		return err
	}
	addr := prefix.Addr().As16()
	bytes := make([]byte, len(addr))
	copy(bytes, addr[:])
	c.AllocatedPrefixes, _, _ = c.AllocatedPrefixes.Insert(bytes, prefix)
	return nil
}

func (c *Calculator) DeleteAllocatedPrefix(prefix netip.Prefix) {
	addr := prefix.Addr().As16()
	bytes := make([]byte, len(addr))
	copy(bytes, addr[:])
	c.AllocatedPrefixes, _, _ = c.AllocatedPrefixes.Delete(bytes)
}

// PrefiInPools tests to see if a prefix is a part of any of the
// pools that have been added to the calculator.
func (c *Calculator) PrefixInPools(prefix netip.Prefix) bool {
	result := false
	c.Pools.Root().Walk(func(k []byte, v interface{}) bool {
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

// NextAvailableSubnet finds the first available subnet of a given mask length
// from a list of subnets and supernets, and fails if none are available.
func (c *Calculator) NextAvailableSubnet(numBits int) (netip.Prefix, error) {
	// For each eligible subnet, walk the tree and determine if the subnet is
	// available for use, and return the first subnet that is available.
	sf := newSubnetFactory(c, numBits)
	defer sf.stop()

	for subnet := range sf.subnetsChan {
		if c.prefixAvailable(subnet) {
			addr := subnet.Addr().As16()
			bytes := make([]byte, len(addr))
			copy(bytes, addr[:])
			c.AllocatedPrefixes, _, _ = c.AllocatedPrefixes.Insert(bytes, subnet)
			return subnet, nil
		}
	}

	return netip.Prefix{}, fmt.Errorf("No eligible subnet with mask /%v found", numBits)
}

// subnetAvailable tests to see if an IPNet is available in an existing tree of subnets.
func (c *Calculator) prefixAvailable(prefix netip.Prefix) bool {
	result := true
	c.AllocatedPrefixes.Root().Walk(func(k []byte, v interface{}) bool {
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
	mode         Mode
	doneChan     chan struct{}
}

func newSubnetFactory(c *Calculator, prefixLength int) *subnetFactory {
	sf := &subnetFactory{
		supernets:    c.Pools,
		prefixLength: prefixLength,
		subnetsChan:  make(chan netip.Prefix),
		mode:         c.Mode,
		doneChan:     make(chan struct{}),
	}
	go sf.run()
	return sf
}

func (sf *subnetFactory) stop() {
	close(sf.doneChan)
}

func (sf *subnetFactory) run() {
	switch sf.mode {
	case ModeV4:
		sf.run4()
	case ModeV6:
		sf.run6()
	default:
		panic("subnetFactory mode unset")
	}
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
