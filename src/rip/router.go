package main

import (
	"fmt"
	"log"
	"net"

	"golang.org/x/net/ipv4"

	"addr"
)

type RipRouter struct {
	done  chan int // close this channel to request end of rip router
	input chan udpInfo
	nets  []*net.IPNet // locally generated networks
	ports []*port      // rip interfaces
	group net.IP       // rip multicast address
}

// rip interface
type port struct {
	name string
	ifi  *net.Interface
	conn *ipv4.PacketConn
}

type udpInfo struct {
	info    []byte
	src     net.IP
	dst     net.IP
	ifIndex int
	ifName  string
}

func NewRipRouter() *RipRouter {

	/*
		proto := "udp"
		hostPort := ":520"

		addr, err1 := net.ResolveUDPAddr(proto, hostPort)
		if err1 != nil {
			log.Printf("NewRipRouter: bad UDP addr=%s/%s: %v", proto, hostPort, err1)
			return nil
		}

		log.Printf("NewRipRouter: reading from: %v", addr)

		conn, err2 := net.ListenUDP(proto, addr)
		if err2 != nil {
			log.Printf("NewRipRouter: listen error addr=%s/%s: %v", proto, hostPort, err2)
			return nil
		}

		input := make(chan udpInfo)

		go udpReader(conn, input)
	*/

	input := make(chan udpInfo)

	r := &RipRouter{done: make(chan int), group: net.IPv4(224, 0, 0, 9)}

	addInterfaces(r, input)

	go func() {
		log.Printf("rip router goroutine started")

		//defer conn.Close()

	LOOP:
		for {
			select {
			case <-r.done:
				// finish requested
				break LOOP
			case u, ok := <-input:
				if !ok {
					log.Printf("rip router: udpReader channel closed")
					break LOOP
				}
				log.Printf("rip router: recv %d bytes from %v to %v on %v", len(u.info), u.src, u.dst, u.ifIndex)
			}
		}

		log.Printf("rip router goroutine finished")
	}()

	return r
}

/*
func udpReader(conn *net.UDPConn, input chan<- udpInfo) {
	buf := make([]byte, 10000)

	defer close(input)

	for {
		n, from, err := conn.ReadFromUDP(buf)
		log.Printf("udpReader: %d bytes from %v: error: %v", n, from, err)
		if err != nil {
			log.Printf("udpReader: error: %v", err)
			break
		}

		// make a copy because we will overwrite buf
		b := make([]byte, n)
		copy(b, buf)

		input <- udpInfo{info: b, addr: *from}
	}
}
*/

func (r *RipRouter) NetAdd(s string) error {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("RipRouter.NetAdd: parse error: addr=[%s]: %v", s, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("RipRouter.NetAdd: bad mask: addr=[%s]: %v", s, err1)
	}
	for _, a := range r.nets {
		if addr.NetEqual(ipnet, a) {
			// found
			return nil
		}
	}
	// not found
	r.nets = append(r.nets, ipnet) // add
	return nil
}

func (r *RipRouter) NetDel(s string) error {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("RipRouter.NetAdd: parse error: addr=[%s]: %v", s, err)
	}
	if err1 := addr.CheckMask(ipnet); err1 != nil {
		return fmt.Errorf("RipRouter.NetDel: bad mask: addr=[%s]: %v", s, err1)
	}
	for i, a := range r.nets {
		if addr.NetEqual(ipnet, a) {
			// found

			last := len(r.nets) - 1
			r.nets[i] = r.nets[last] // overwrite position with last pointer
			r.nets[last] = nil       // free last pointer for garbage collection
			r.nets = r.nets[:last]   // shrink

			return nil
		}
	}
	// not found
	return nil
}

func addInterfaces(r *RipRouter, input chan<- udpInfo) {
	ifList, err1 := net.Interfaces()
	if err1 != nil {
		log.Printf("NewRipRouter: could not find local interfaces: %v", err1)
	}
	for _, i := range ifList {
		if err := r.InterfaceAdd(i.Name); err != nil {
			log.Printf("NewRipRouter: error adding interface '%s': %v", i.Name, err)
		}
	}
}

func (r *RipRouter) InterfaceAdd(s string) error {
	//log.Printf("RipRouter.InterfaceAdd: %s", s)

	for _, p := range r.ports {
		if s == p.name {
			return fmt.Errorf("RipRouter.InterfaceAdd: interface '%s' exists", s)
		}
	}

	iface, err1 := net.InterfaceByName(s)
	if err1 != nil {
		return err1
	}

	addrList, err2 := iface.Addrs()
	if err2 != nil {
		return err2
	}

	for _, a := range addrList {
		addr, _, err3 := net.ParseCIDR(a.String())
		if err3 != nil {
			log.Printf("RipRouter.InterfaceAdd: parse CIDR error for '%s' on '%s': %v", addr, s, err3)
			continue
		}
		if addr.IsMulticast() {
			// bind only to unicast addresses
			continue
		}
		if err := r.Join(iface, addr); err != nil {
			log.Printf("RipRouter.InterfaceAdd: join error for '%s' on '%s': %v", addr, s, err)
		}
	}

	return nil
}

func (r *RipRouter) Join(iface *net.Interface, addr net.IP) error {
	proto := "udp"
	var a string
	if addr.To4() == nil {
		// IPv6
		a = fmt.Sprintf("[%s]", addr.String())
	} else {
		// IPv4
		a = addr.String()
	}

	hostPort := fmt.Sprintf("%s:520", a) // rip multicast port

	//log.Printf("Join: %s %s %s", iface.Name, proto, hostPort)

	// open socket (connection)
	conn, err2 := net.ListenPacket(proto, hostPort)
	if err2 != nil {
		return fmt.Errorf("RipRouter.InterfaceAdd: net.ListenPacket: %s/%s error: %v", proto, hostPort, err2)
	}

	// join multicast address
	pc := ipv4.NewPacketConn(conn)
	if err := pc.JoinGroup(iface, &net.UDPAddr{IP: r.group}); err != nil {
		conn.Close()
		return fmt.Errorf("RipRouter.InterfaceAdd: join error: %v", err)
	}

	// request control messages
	if err := pc.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		// warning only
		log.Printf("RipRouter.InterfaceAdd: control message flags error: %v", err)
	}

	newPort := &port{name: iface.Name, ifi: iface, conn: pc}

	r.ports = append(r.ports, newPort)

	go udpReader(pc, r.input, iface.Name, addr.String())

	return nil
}

func udpReader(c *ipv4.PacketConn, input chan<- udpInfo, ifname, ifaddr string) {

	log.Printf("udpReader: reading from '%s' on '%s'", ifaddr, ifname)

	defer c.Close()

	buf := make([]byte, 10000)

	for {
		n, cm, _, err := c.ReadFrom(buf)
		if err != nil {
			log.Printf("udpReader: ReadFrom: error %v", err)
			break
		}

		// make a copy because we will overwrite buf
		b := make([]byte, n)
		copy(b, buf)

		log.Printf("udpReader: recv %d bytes from %s to %s on %s", n, cm.Src, cm.Dst, ifname)

		input <- udpInfo{info: b, src: cm.Src, dst: cm.Dst, ifIndex: cm.IfIndex, ifName: ifname}
	}

	log.Printf("udpReader: exiting '%s'", ifname)
}

func (r *RipRouter) InterfaceDel(s string) error {
	log.Printf("RipRouter.InterfaceDel: %s", s)

	for _, p := range r.ports {
		if s == p.name {
			// found interface

			if err := p.conn.LeaveGroup(p.ifi, &net.UDPAddr{IP: r.group}); err != nil {
				// warning only
				log.Printf("RipRouter.InterfaceDel: leave group error: %v", err)
			}

			p.conn.Close() // should kill reader goroutine

			return nil
		}
	}

	return fmt.Errorf("RipRouter.InterfaceDel: interface '%s' not found", s)
}
