package main

import (
	"fmt"
	"math/big"
	"net"
)

/*
func CalculateCIDRSubnets(parentCIDR string, numSubnets int) ([]string, error) {
	// Parse the parent CIDR into an IPNet struct
	_, parentIPNet, err := net.ParseCIDR(parentCIDR)
	if err != nil {
		return nil, err
	}

	// Calculate the number of bits needed to represent the required subnets
	bitsNeeded := 0
	for i := numSubnets; i > 1; i /= 2 {
		bitsNeeded++
	}

	// Calculate the new subnet mask
	newMask := parentIPNet.Mask[len(parentIPNet.Mask)-4:]
	for i := 31; i >= 0; i-- {
		if bitsNeeded > 0 {
			newMask[i/8] |= 1 << uint(7-(i%8))
			bitsNeeded--
		}
	}

	// Create new CIDR subnets
	subnets := make([]string, numSubnets)
	for i := 0; i < numSubnets; i++ {
		newIPNet := &net.IPNet{
			IP:   parentIPNet.IP,
			Mask: net.CIDRMask(len(newMask)*8, len(newMask)*8),
		}
		subnets[i] = newIPNet.String()
		parentIPNet.IP = nextSubnetIP(parentIPNet.IP, newMask)
	}

	return subnets, nil
}

func nextSubnetIP(ip net.IP, mask []byte) net.IP {
	ipLen := len(ip)
	nextIP := make(net.IP, ipLen)
	copy(nextIP, ip)

	// Increment the IP address to the next subnet
	for i := ipLen - 1; i >= 0; i-- {
		nextIP[i]++
		if nextIP[i] != 0 {
			break
		}
	}

	return nextIP
}
*/

func CalculateCIDRSubnets(parentCIDR string, numSubnets int, bitsToMask int) ([]string, error) {
	// Parse the parent CIDR into an IPNet struct
	_, parentIPNet, err := net.ParseCIDR(parentCIDR)
	if err != nil {
		return nil, err
	}

	// Calculate the new subnet mask
	maskLen := len(parentIPNet.IP) * 8
	if bitsToMask > maskLen {
		return nil, fmt.Errorf("Bits to mask exceeds the available bits in the parent CIDR")
	}

	// Create new CIDR subnets
	subnets := make([]string, numSubnets)
	mask := net.CIDRMask(bitsToMask, maskLen)
	subnetSize := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(maskLen-bitsToMask)), nil)

	for i := 0; i < numSubnets; i++ {
		subnets[i] = fmt.Sprintf("%s/%d", parentIPNet.IP, bitsToMask)
		parentIPNet.IP = nextSubnetIP(parentIPNet.IP, mask, subnetSize)
	}

	return subnets, nil
}

func nextSubnetIP(ip net.IP, mask net.IPMask, subnetSize *big.Int) net.IP {
	ipLen := len(ip)
	nextIP := make(net.IP, ipLen)
	copy(nextIP, ip)

	// Calculate the next subnet IP by adding the subnet size
	bigIP := new(big.Int).SetBytes(nextIP)
	bigIP.Add(bigIP, subnetSize)
	nextIPBytes := bigIP.Bytes()

	// Pad the bytes to match the IP length
	copy(nextIP[ipLen-len(nextIPBytes):], nextIPBytes)

	return nextIP
}
