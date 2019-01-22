//
// Copyright (c) 2017 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/mdlayher/vsock"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
)

var (
	channelExistMaxTries   = 200
	channelExistWaitTime   = 50 * time.Millisecond
	channelCloseTimeout    = 5 * time.Second
	isAFVSockSupportedFunc = isAFVSockSupported
)

type channel interface {
	setup() error
	wait() error
	listen() (net.Listener, error)
	teardown() error
}

type unixDomainSocket struct {
	// Socket address
	addr *url.URL

	// Needs to be closed to avoid leakage
	listener *net.UnixListener

	// Used to determine when the Yamux stream has closed
	waitCh <-chan struct{}
}

func (u *unixDomainSocket) setup() error {
	agentLog.Infof("DEBUG: unixDomainSocket.setup:")
	return nil
}

func (u *unixDomainSocket) wait() error {
	agentLog.Infof("DEBUG: unixDomainSocket.wait:")
	return nil
}

func (u *unixDomainSocket) listen() (net.Listener, error) {
	agentLog.Infof("DEBUG: unixDomainSocket.listen:")

	unixAddr := &net.UnixAddr{
		Name: u.addr.Path,
		Net:  u.addr.Scheme,
	}

	var err error

	u.listener, err = net.ListenUnix(u.addr.Scheme, unixAddr)
	if err != nil {
		return nil, err
	}

	conn, err := u.listener.Accept()
	if err != nil {
		return nil, err
	}

	session, err := setupYamux(conn)
	if err != nil {
		return nil, err
	}

	u.waitCh = session.CloseChan()

	return session, nil
}

func (u *unixDomainSocket) teardown() error {
	agentLog.Infof("DEBUG: unixDomainSocket.teardown:")

	err := waitForYamuxStop(u.waitCh)
	if err != nil {
		return err
	}

	return u.listener.Close()
}

func newUnixDomainSocketChannel(socketPath string) (*unixDomainSocket, error) {
	if socketPath == "" {
		return &unixDomainSocket{}, errors.New("empty socket path")

	}

	addr, err := url.Parse(socketPath)
	if err != nil {
		return nil, err
	}

	if addr.Scheme != "" && addr.Scheme != "unix" {
		return nil, fmt.Errorf("invalid socket address scheme: %q", socketPath)
	}

	if err := os.Remove(addr.Path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return &unixDomainSocket{
		addr: addr,
	}, nil
}

// Creates a new channel to communicate the agent with the proxy or shim.
// The runtime hot plugs a serial port or a vsock PCI depending of the configuration
// file and if the host has support for vsocks. newChannel iterates in a loop looking
// for the serial port or vsock device.
// The timeout is defined by channelExistMaxTries and channelExistWaitTime and it
// can be calculated by using the following operation:
// (channelExistMaxTries * channelExistWaitTime) / 1000 = timeout in seconds
// If there are neither vsocks nor serial ports, an error is returned.
func newChannel(channelPath string) (channel, error) {
	var serialErr error
	var vsockErr error
	var vSockSupported bool
	var serialPath string

	c, domainErr := newUnixDomainSocketChannel(channelPath)

	if domainErr == nil {
		agentLog.WithFields(logrus.Fields{"channel-path": c.addr, "channel-type": "unix"}).Info("Found channel")
		return c, nil
	} else {
		agentLog.WithError(domainErr).WithField("invalid-channel-path", channelPath).Info("failed to create unix domain channel")
	}

	for i := 0; i < channelExistMaxTries; i++ {
		// check vsock path
		if _, err := os.Stat(vSockDevPath); err == nil {
			agentLog.Infof("DEBUG: newChannel: file %v exists", vSockDevPath)

			if vSockSupported, vsockErr = isAFVSockSupportedFunc(); vSockSupported && vsockErr == nil {
				agentLog.WithField("channel-type", "vsock").Info("Found channel")
				return &vSockChannel{}, nil
			}
		}

		// Check serial port path
		if serialPath, serialErr = findVirtualSerialPath(serialChannelName); serialErr == nil {
			agentLog.WithField("channel-type", "serial").Info("Found channel")
			return &serialChannel{serialPath: serialPath}, nil
		}

		time.Sleep(channelExistWaitTime)
	}

	if domainErr != nil {
		agentLog.WithError(serialErr).Error("Unix Domain socket not found")
	}

	if serialErr != nil {
		agentLog.WithError(serialErr).Error("Serial port not found")
	}

	if vsockErr != nil {
		agentLog.WithError(vsockErr).Error("VSock not found")
	}

	return nil, fmt.Errorf("No available channels found")
}

type vSockChannel struct {
}

func (c *vSockChannel) setup() error {
	return nil
}

func (c *vSockChannel) wait() error {
	return nil
}

func (c *vSockChannel) listen() (net.Listener, error) {
	l, err := vsock.Listen(vSockPort)
	if err != nil {
		return nil, err
	}

	return l, nil
}

func (c *vSockChannel) teardown() error {
	return nil
}

type serialChannel struct {
	serialPath string
	serialConn *os.File
	waitCh     <-chan struct{}
}

func (c *serialChannel) setup() error {
	// Open serial channel.
	file, err := os.OpenFile(c.serialPath, os.O_RDWR, os.ModeDevice)
	if err != nil {
		return err
	}

	c.serialConn = file

	return nil
}

func (c *serialChannel) wait() error {
	var event unix.EpollEvent
	var events [1]unix.EpollEvent

	fd := c.serialConn.Fd()
	if fd <= 0 {
		return fmt.Errorf("serial port IO closed")
	}

	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return err
	}
	defer unix.Close(epfd)

	// EPOLLOUT: Writable when there is a connection
	// EPOLLET: Edge trigger as EPOLLHUP is always on when there is no connection
	// 0xffffffff: EPOLLET is negative and cannot fit in uint32 in golang
	event.Events = unix.EPOLLOUT | unix.EPOLLET&0xffffffff
	event.Fd = int32(fd)
	if err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(fd), &event); err != nil {
		return err
	}
	defer unix.EpollCtl(epfd, unix.EPOLL_CTL_DEL, int(fd), nil)

	for {
		nev, err := unix.EpollWait(epfd, events[:], -1)
		if err != nil {
			return err
		}

		for i := 0; i < nev; i++ {
			ev := events[i]
			if ev.Fd == int32(fd) {
				agentLog.WithField("events", ev.Events).Debug("New serial channel event")
				if ev.Events&unix.EPOLLOUT != 0 {
					return nil
				}
				if ev.Events&unix.EPOLLERR != 0 {
					return fmt.Errorf("serial port IO failure")
				}
				if ev.Events&unix.EPOLLHUP != 0 {
					continue
				}
			}
		}
	}

	// Never reach here
}

// yamuxWriter is a type responsible for logging yamux messages to the agent
// log.
type yamuxWriter struct {
}

// Write implements the Writer interface for the yamuxWriter.
func (yw yamuxWriter) Write(bytes []byte) (int, error) {
	message := string(bytes)

	l := len(message)

	// yamux messages are all warnings and errors
	agentLog.WithField("component", "yamux").Warn(message)

	return l, nil
}

func setupYamux(conn io.ReadWriteCloser) (*yamux.Session, error) {
	agentLog.Infof("DEBUG: setupYamux: conn: +%v", conn)

	config := yamux.DefaultConfig()
	// yamux client runs on the proxy side, sometimes the client is
	// handling other requests and it's not able to response to the
	// ping sent by the server and the communication is closed. To
	// avoid any IO timeouts in the communication between agent and
	// proxy, keep alive should be disabled.
	config.EnableKeepAlive = false
	config.LogOutput = yamuxWriter{}

	// Initialize Yamux server.
	session, err := yamux.Server(conn, config)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (c *serialChannel) listen() (net.Listener, error) {
	session, err := setupYamux(c.serialConn)
	if err != nil {
		return nil, err
	}

	c.waitCh = session.CloseChan()

	return session, nil
}

// waitForYamuxStop waits for the session to be fully shutdown
func waitForYamuxStop(ch <-chan struct{}) error {
	if ch == nil {
		return nil
	}

	t := time.NewTimer(channelCloseTimeout)
	select {
	case <-ch:
		t.Stop()
	case <-t.C:
		return errors.New("timeout waiting for yamux channel to close")
	}

	return nil

}

func (c *serialChannel) teardown() error {
	return waitForYamuxStop(c.waitCh)
}

func isAFVSockSupported() (bool, error) {
	fd, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		// This case is valid. It means AF_VSOCK is not a supported
		// domain on this system.
		if err == unix.EAFNOSUPPORT {
			return false, nil
		}

		return false, err
	}

	if err := unix.Close(fd); err != nil {
		return true, err
	}

	return true, nil
}

func findVirtualSerialPath(serialName string) (string, error) {
	dir, err := os.Open(virtIOPath)
	if err != nil {
		return "", err
	}

	defer dir.Close()

	ports, err := dir.Readdirnames(0)
	if err != nil {
		return "", err
	}

	for _, port := range ports {
		path := filepath.Join(virtIOPath, port, "name")
		content, err := ioutil.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				agentLog.WithField("file", path).Debug("Skip parsing of non-existent file")
				continue
			}
			return "", err
		}

		if strings.Contains(string(content), serialName) == true {
			return filepath.Join(devRootPath, port), nil
		}
	}

	return "", grpcStatus.Errorf(codes.NotFound, "Could not find virtio port %s", serialName)
}
