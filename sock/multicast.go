package sock

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/net/ipv4"
)

type MulticastSock struct {
	P *ipv4.PacketConn
	U *net.UDPConn
}

func NewUDPConn(laddr *net.UDPAddr, ifname string) (*net.UDPConn, error) {

	/*
		if laddr == nil {
			laddr = &net.UDPAddr{IP: net.IPv4zero, Port: 0}
		}

		s, err1 := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
		if err1 != nil {
			return nil, fmt.Errorf("MulticastListener: could not create socket(laddr=%v,ifname=%s): %v", laddr, ifname, err1)
		}
		if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			syscall.Close(s)
			return nil, fmt.Errorf("MulticastListener: could not set reuse addr socket(laddr=%v,ifname=%s): %v", laddr, ifname, err)
		}
		if ifname != "" {
			if err := syscall.SetsockoptString(s, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifname); err != nil {
				syscall.Close(s)
				return nil, fmt.Errorf("MulticastListener: could not bind to device socket(laddr=%v, ifname=%s): %v", laddr, ifname, err)
			}
		}

		lsa := syscall.SockaddrInet4{Port: laddr.Port}
		copy(lsa.Addr[:], laddr.IP.To4())

		if err := syscall.Bind(s, &lsa); err != nil {
			syscall.Close(s)
			return nil, fmt.Errorf("MulticastListener: could not bind socket to address %v: %v", laddr, err)
		}
		f := os.NewFile(uintptr(s), "")
		c, err2 := net.FilePacketConn(f)
		f.Close()
		if err2 != nil {
			syscall.Close(s)
			return nil, fmt.Errorf("MulticastListener: could not get packet connection for socket(laddr=%v,ifname=%s): %v", laddr, ifname, err2)
		}
		u := c.(*net.UDPConn)

		return u, nil
	*/

	c, err := udpConn(laddr, ifname)
	if err != nil {
		return nil, err
	}

	u := c.(*net.UDPConn)

	return u, nil
}

func udpConn(laddr *net.UDPAddr, ifname string) (net.PacketConn, error) {
	if laddr == nil {
		laddr = &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	}

	s, err1 := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err1 != nil {
		return nil, fmt.Errorf("MulticastListener: could not create socket(laddr=%v,ifname=%s): %v", laddr, ifname, err1)
	}
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("MulticastListener: could not set reuse addr socket(laddr=%v,ifname=%s): %v", laddr, ifname, err)
	}
	if ifname != "" {
		if err := syscall.SetsockoptString(s, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifname); err != nil {
			syscall.Close(s)
			return nil, fmt.Errorf("MulticastListener: could not bind to device socket(laddr=%v, ifname=%s): %v", laddr, ifname, err)
		}
	}

	lsa := syscall.SockaddrInet4{Port: laddr.Port}
	copy(lsa.Addr[:], laddr.IP.To4())

	if err := syscall.Bind(s, &lsa); err != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("MulticastListener: could not bind socket to address %v: %v", laddr, err)
	}
	f := os.NewFile(uintptr(s), "")
	c, err2 := net.FilePacketConn(f)
	f.Close()
	if err2 != nil {
		syscall.Close(s)
		return nil, fmt.Errorf("MulticastListener: could not get packet connection for socket(laddr=%v,ifname=%s): %v", laddr, ifname, err2)
	}

	return c, nil
}

func MulticastListener(port int, ifname string) (*MulticastSock, error) {

	/*
		s, err1 := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
		if err1 != nil {
			return nil, fmt.Errorf("MulticastListener: could not create socket(port=%d,ifname=%s): %v", port, ifname, err1)
		}
		if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			syscall.Close(s)
			return nil, fmt.Errorf("MulticastListener: could not set reuse addr socket(port=%d,ifname=%s): %v", port, ifname, err)
		}
		if ifname != "" {
			if err := syscall.SetsockoptString(s, syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, ifname); err != nil {
				syscall.Close(s)
				return nil, fmt.Errorf("MulticastListener: could not bind to device socket(port=%d,ifname=%s): %v", port, ifname, err)
			}
		}

		bindAddr := net.IP(net.IPv4(0, 0, 0, 0))
		lsa := syscall.SockaddrInet4{Port: port}
		copy(lsa.Addr[:], bindAddr.To4())

		if err := syscall.Bind(s, &lsa); err != nil {
			syscall.Close(s)
			return nil, fmt.Errorf("MulticastListener: could not bind socket to address %v,%d: %v", bindAddr, port, err)
		}
		f := os.NewFile(uintptr(s), "")
		c, err2 := net.FilePacketConn(f)
		f.Close()
		if err2 != nil {
			syscall.Close(s)
			return nil, fmt.Errorf("MulticastListener: could not get packet connection for socket(port=%d,ifname=%s): %v", port, ifname, err2)
		}
		u := c.(*net.UDPConn)
		p := ipv4.NewPacketConn(c)

		if err := p.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
			return nil, fmt.Errorf("MulticastListener: could not set control message flags: %v", err)
		}

		return &MulticastSock{P: p, U: u}, nil
	*/

	c, err := udpConn(&net.UDPAddr{IP: net.IPv4zero, Port: port}, ifname)
	if err != nil {
		return nil, err
	}

	u := c.(*net.UDPConn)
	p := ipv4.NewPacketConn(c)

	if err := p.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		return nil, fmt.Errorf("MulticastListener: could not set control message flags: %v", err)
	}

	return &MulticastSock{P: p, U: u}, nil
}

func Join(sock *MulticastSock, group net.IP, ifname string) error {
	ifi, err1 := net.InterfaceByName(ifname)
	if err1 != nil {
		return fmt.Errorf("Join: could not find interface %s: %v", ifname, err1)
	}

	if err := sock.P.JoinGroup(ifi, &net.UDPAddr{IP: group}); err != nil {
		return fmt.Errorf("Join: could not join group %v on interface %s: %v", group, ifname, err)
	}

	return nil
}

func Leave(sock *MulticastSock, group net.IP, ifi *net.Interface) error {
	if err := sock.P.LeaveGroup(ifi, &net.UDPAddr{IP: group}); err != nil {
		return fmt.Errorf("Leave: could not leave group %v on interface %s: %v", group, ifi.Name, err)
	}

	return nil
}

func Close(sock *MulticastSock) {
	sock.P.Close()
	sock.U.Close()
	sock.P = nil
	sock.U = nil
}
