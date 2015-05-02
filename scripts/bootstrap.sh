#!/bin/bash
set -e

# configure link layer
netrack -d s1 ip link set s1-eth1 --address 0050.5600.0001
netrack -d s1 ip link set s1-eth2 --address 0050.5600.0002
netrack -d s1 ip link set s1-eth3 --address 0050.5600.0003

# configure network addresses
netrack -d s1 ip addr add 10.0.1.254/24 --device s1-eth1
netrack -d s1 ip addr add 10.0.2.254/24 --device s1-eth2
netrack -d s1 ip addr add 10.0.3.254/24 --device s1-eth3
