package subnetcalculator

import (
	"fmt"
	"math"
	"math/big"
	"net"

	iradix "github.com/hashicorp/go-immutable-radix"
)

// SubnetCalculator stores radix trees of supernets and subnets.
type SubnetCalculator struct {
	Supernets *iradix.Tree
	Subnets   *iradix.Tree
}

// New creates a new SubnetCalculator from a list of supernets and subnets.
func New(supernets, subnets []net.IPNet) SubnetCalculator {
	subnetsTree := iradix.New()
	for _, subnet := range subnets {
		subnetsTree, _, _ = subnetsTree.Insert(subnet.IP, subnet)
	}
	supernetsTree := iradix.New()
	for _, supernet := range supernets {
		supernetsTree, _, _ = supernetsTree.Insert(supernet.IP, supernet)
	}
	return SubnetCalculator{
		Supernets: supernetsTree,
		Subnets:   subnetsTree,
	}
}

// NextAvailableSubnet finds the first available subnet of a given mask length
// from a list of subnets and supernets, and fails if none are available.
func (sc *SubnetCalculator) NextAvailableSubnet(mask net.IPMask) (net.IPNet, error) {
	// generate all eligible subnets with <mask> length from provided supernets
	eligible := sc.generateEligibleIPNets(mask)
	// for each eligible subnet, walk the tree and determine if the subnet is
	// available for use, and return the first subnet that is available.
	for _, subnet := range eligible {
		if sc.subnetAvailable(subnet) {
			sc.Subnets, _, _ = sc.Subnets.Insert(subnet.IP, subnet)
			return subnet, nil
		}
	}
	ones, _ := mask.Size()
	return net.IPNet{}, fmt.Errorf("No eligible subnet with mask /%v found", ones)
}

// subnetAvailable tests to see if an IPNet is available in an existing tree of subnets.
func (sc *SubnetCalculator) subnetAvailable(ipnet net.IPNet) bool {
	result := true
	sc.Subnets.Root().Walk(func(k []byte, v interface{}) bool {
		n, ok := v.(net.IPNet)
		if !ok {
			panic("unexpected node type found in radix tree")
		}
		ones, _ := n.Mask.Size()
		// fmt.Printf("net: %s   ones: %d   bits: %d\n", currentIPNet.String(), ones, bits)
		if ones <= 16 {
			return false
		}
		if n.Contains(ipnet.IP) {
			result = false
			return true
		}
		if ipnet.Contains(n.IP) {
			result = false
			return true
		}
		return false
	})
	return result
}

// PercentUsed calculates what percentage of the Supernets have been
// used by the Subnets.
func (sc *SubnetCalculator) PercentUsed() float64 {
	var totalAvailable, totalUsed float64

	sc.Supernets.Root().Walk(func(k []byte, v interface{}) bool {
		n, ok := v.(net.IPNet)
		if !ok {
			panic("unexpected node type found in radix tree")
		}
		ones, bits := n.Mask.Size()
		totalAvailable = totalAvailable + math.Exp2(float64(bits-ones))
		return false
	})

	sc.Subnets.Root().Walk(func(k []byte, v interface{}) bool {
		n, ok := v.(net.IPNet)
		if !ok {
			panic("unexpected node type found in radix tree")
		}
		ones, bits := n.Mask.Size()
		totalUsed = totalUsed + math.Exp2(float64(bits-ones))
		return false
	})

	return (totalUsed / totalAvailable) * float64(100)
}

// generateEligibleIPNets calculates all possible subnets with a given mask
// from all available supernets.
func (sc *SubnetCalculator) generateEligibleIPNets(mask net.IPMask) []net.IPNet {
	result := []net.IPNet{}
	sc.Supernets.Root().Walk(func(k []byte, v interface{}) bool {
		n, ok := v.(net.IPNet)
		if !ok {
			panic("unexpected node type found in radix tree")
		}
		netOnes, _ := n.Mask.Size()
		maskOnes, _ := mask.Size()
		if netOnes > maskOnes {
			return false
		}
		// Starting network is base network of CIDR, with new calculated mask
		baseNet := net.IPNet{IP: n.IP, Mask: mask}
		numSubnets := int(math.Exp2(float64(maskOnes - netOnes)))
		for i := 0; i < numSubnets; i++ {
			nextNet := getSubnet(baseNet, i)
			result = append(result, nextNet)
		}
		return false
	})
	return result
}

// getSubnet calculates the nth subnet up from a starting subnet.
func getSubnet(ip net.IPNet, n int) net.IPNet {
	// Get lowest mask bit position
	l, _ := ip.Mask.Size()
	l = 32 - l
	// Calculate integer value times multiplier to get step up from base network
	step := int64(math.Exp2(float64(l))) * int64(n)
	// Convert base network IP to large integer
	baseNetInt := new(big.Int)
	baseNetInt.SetBytes(ip.IP)
	// Add step value to base network to get new network
	newNetInt := big.NewInt(baseNetInt.Int64() + step)
	// Convert new network int to net.IP
	newNet := net.IP(newNetInt.Bytes())
	// Construct new net.IPNet from new network and mask
	return net.IPNet{IP: newNet, Mask: ip.Mask}
}
