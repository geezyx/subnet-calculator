package subnetcalculator

import (
	"errors"
	"fmt"
	"net/netip"

	iradix "github.com/hashicorp/go-immutable-radix"
)

// SubnetCalculator stores radix trees of supernets and subnets.
type SubnetCalculator struct {
	Supernets *iradix.Tree
	Subnets   *iradix.Tree
	Mode      Mode
}

type Mode int

const (
	ModeV4 Mode = iota + 1
	ModeV6
)

// New creates a new SubnetCalculator from a list of supernets and subnets.
func New(supernets, subnets []netip.Prefix) (SubnetCalculator, error) {
	if len(supernets) == 0 {
		return SubnetCalculator{}, errors.New("At least one CIDR block must be provided.")
	}

	var mode Mode = 0

	supernetsTree := iradix.New()
	for _, supernet := range supernets {
		switch {
		case mode == 0 && supernet.Addr().Is4():
			mode = ModeV4
		case mode == 0 && supernet.Addr().Is6():
			mode = ModeV6
		case mode == ModeV4 && !supernet.Addr().Is4():
			return SubnetCalculator{}, errors.New("Mix of IPv4 and IPv6 CIDRs detected, all CIDRs must be of the same type.")
		case mode == ModeV6 && !supernet.Addr().Is6():
			return SubnetCalculator{}, errors.New("Mix of IPv4 and IPv6 CIDRs detected, all CIDRs must be of the same type.")
		}
		addr := supernet.Addr().As16()
		bytes := make([]byte, len(addr))
		copy(bytes, addr[:])
		supernetsTree, _, _ = supernetsTree.Insert(bytes, supernet)
	}
	subnetsTree := iradix.New()
	for _, subnet := range subnets {
		switch {
		case mode == ModeV4 && !subnet.Addr().Is4():
			return SubnetCalculator{}, errors.New("Mix of IPv4 and IPv6 CIDRs detected, all CIDRs must be of the same type.")
		case mode == ModeV6 && !subnet.Addr().Is6():
			return SubnetCalculator{}, errors.New("Mix of IPv4 and IPv6 CIDRs detected, all CIDRs must be of the same type.")
		}
		addr := subnet.Addr().As16()
		bytes := make([]byte, len(addr))
		copy(bytes, addr[:])
		subnetsTree, _, _ = subnetsTree.Insert(bytes, subnet)
	}

	return SubnetCalculator{
		Supernets: supernetsTree,
		Subnets:   subnetsTree,
		Mode:      mode,
	}, nil
}

// NextAvailableSubnet finds the first available subnet of a given mask length
// from a list of subnets and supernets, and fails if none are available.
func (sc *SubnetCalculator) NextAvailableSubnet(numBits int) (netip.Prefix, error) {
	// for each eligible subnet, walk the tree and determine if the subnet is
	// available for use, and return the first subnet that is available.

	sf := newSubnetFactory(sc, numBits)
	defer sf.stop()

	for subnet := range sf.subnetsChan {
		if sc.subnetAvailable(subnet) {
			addr := subnet.Addr().As16()
			bytes := make([]byte, len(addr))
			copy(bytes, addr[:])
			sc.Subnets, _, _ = sc.Subnets.Insert(bytes, subnet)
			return subnet, nil
		}
	}

	return netip.Prefix{}, fmt.Errorf("No eligible subnet with mask /%v found", numBits)
}

// subnetAvailable tests to see if an IPNet is available in an existing tree of subnets.
func (sc *SubnetCalculator) subnetAvailable(ipnet netip.Prefix) bool {
	result := true
	sc.Subnets.Root().Walk(func(k []byte, v interface{}) bool {
		n, ok := v.(netip.Prefix)
		if !ok {
			panic("unexpected node type found in radix tree")
		}
		// ones, _ := n.Mask.Size()
		// // fmt.Printf("net: %s   ones: %d   bits: %d\n", currentIPNet.String(), ones, bits)
		// if ones <= 16 {
		// 	return false
		// }
		if n.Contains(ipnet.Addr()) {
			result = false
			return true
		}
		if ipnet.Contains(n.Addr()) {
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

func newSubnetFactory(sc *SubnetCalculator, prefixLength int) *subnetFactory {
	sf := &subnetFactory{
		supernets:    sc.Supernets,
		prefixLength: prefixLength,
		subnetsChan:  make(chan netip.Prefix),
		mode:         sc.Mode,
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
