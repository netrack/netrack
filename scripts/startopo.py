#!/usr/bin/env python2

import mininet
import mininet.net
import mininet.topo
import mininet.node
import mininet.log
import mininet.cli

class StarTopo(mininet.topo.Topo):
    def __init__(self, **kwargs):
        super(StarTopo, self).__init__(**kwargs)

        s1 = self.addSwitch("s1", protocols="OpenFlow13")

        h1 = self.addHost("h1", ip="10.0.1.1/24")
        h2 = self.addHost("h2", ip="10.0.2.1/24")
        h3 = self.addHost("h3", ip="10.0.3.1/24")

        self.addLink(s1, h1, port1=1)
        self.addLink(s1, h2, port1=2)
        self.addLink(s1, h3, port1=3)

def set_default_route(network):
    for host in network.hosts:
        host.cmd("ip route add default via 10.0.{0}.254 dev {1}-eth0".format(
            host.name.replace("h", ""), host.name))

def main():
    mininet.log.setLogLevel("info")

    net = mininet.net.Mininet(topo=StarTopo())
    net.addController(ip="192.168.0.100",
        controller=mininet.node.RemoteController)

    set_default_route(net)

    net.start()
    mininet.cli.CLI(net)
    net.stop()

if __name__ == "__main__":
    main()
