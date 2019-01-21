package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	kataclient "github.com/kata-containers/agent/protocols/client"
	"github.com/kata-containers/agent/protocols/grpc"
	"github.com/sirupsen/logrus"
	golangGrpc "google.golang.org/grpc"
)

type reqFunc func(context.Context, interface{}, ...golangGrpc.CallOption) (interface{}, error)

type kataClient struct {
	// Address used to connect to kata-agent.
	address string

	// Connection to the agent.
	conn *kataclient.AgentClient

	// Table of gRPC handlers.
	handlers map[string]reqFunc

	// Keep connection alive if true.
	keepAlive bool

	enableYamux bool

	// The logger to use
	log *logrus.Entry

	ctx context.Context
}

func newKataClient(ctx context.Context, logger *logrus.Entry, address string, keepAlive, enableYamux bool) (*kataClient, error) {
	if address == "" {
		return nil, errors.New("missing address")
	}

	if logger == nil {
		return nil, errors.New("need logger")
	}

	c := &kataClient{
		address:     address,
		keepAlive:   keepAlive,
		enableYamux: enableYamux,
		log:         logger,
		ctx:         ctx,
	}

	c.logger().Info(c)

	return c, nil
}

func (c *kataClient) logger() *logrus.Entry {
	return c.log.WithField("subsystem", "grpc")
}

func (c *kataClient) connect() error {
	var err error

	if c.conn != nil {
		if c.keepAlive {
			// already connected
			return nil
		}

		return fmt.Errorf("BUG: found existing connection bug keepAlive not set")
	}

	c.conn, err = kataclient.NewAgentClient(c.ctx,
		c.address,
		c.enableYamux)
	if err != nil {
		return err
	}

	c.installHandlers()

	return nil
}

func (c *kataClient) disconnect() error {
	if c.conn == nil {
		return errors.New("no connection")
	}

	err := c.conn.Close()

	// Reset
	c.conn = nil
	c.keepAlive = false
	c.handlers = nil

	return err
}

func (c *kataClient) installHandlers() {
	c.handlers = make(map[string]reqFunc)

	c.handlers["grpc.CheckRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		timeout := time.Duration(grpcTimeoutSecs) * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return c.conn.Check(ctx, req.(*grpc.CheckRequest), opts...)
	}
	c.handlers["grpc.CloseStdinRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.CloseStdin(ctx, req.(*grpc.CloseStdinRequest), opts...)
	}
	c.handlers["grpc.CopyFileRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.CopyFile(ctx, req.(*grpc.CopyFileRequest), opts...)
	}
	c.handlers["grpc.CreateContainerRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.CreateContainer(ctx, req.(*grpc.CreateContainerRequest), opts...)
	}
	c.handlers["grpc.CreateSandboxRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.CreateSandbox(ctx, req.(*grpc.CreateSandboxRequest), opts...)
	}
	c.handlers["grpc.DestroySandboxRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.DestroySandbox(ctx, req.(*grpc.DestroySandboxRequest), opts...)
	}
	c.handlers["grpc.ExecProcessRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.ExecProcess(ctx, req.(*grpc.ExecProcessRequest), opts...)
	}
	c.handlers["grpc.GuestDetailsRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.GetGuestDetails(ctx, req.(*grpc.GuestDetailsRequest), opts...)
	}
	c.handlers["grpc.ListInterfacesRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.ListInterfaces(ctx, req.(*grpc.ListInterfacesRequest), opts...)
	}
	c.handlers["grpc.ListProcessesRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.ListProcesses(ctx, req.(*grpc.ListProcessesRequest), opts...)
	}
	c.handlers["grpc.ListRoutesRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.ListRoutes(ctx, req.(*grpc.ListRoutesRequest), opts...)
	}
	c.handlers["grpc.OnlineCPUMemRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.OnlineCPUMem(ctx, req.(*grpc.OnlineCPUMemRequest), opts...)
	}
	c.handlers["grpc.PauseContainerRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.PauseContainer(ctx, req.(*grpc.PauseContainerRequest), opts...)
	}
	c.handlers["grpc.ResumeContainerRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.ResumeContainer(ctx, req.(*grpc.ResumeContainerRequest), opts...)
	}
	c.handlers["grpc.RemoveContainerRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.RemoveContainer(ctx, req.(*grpc.RemoveContainerRequest), opts...)
	}
	c.handlers["grpc.ReseedRandomDevRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.ReseedRandomDev(ctx, req.(*grpc.ReseedRandomDevRequest), opts...)
	}
	c.handlers["grpc.SetGuestDateTimeRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.SetGuestDateTime(ctx, req.(*grpc.SetGuestDateTimeRequest), opts...)
	}
	c.handlers["grpc.SignalProcessRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.SignalProcess(ctx, req.(*grpc.SignalProcessRequest), opts...)
	}
	c.handlers["grpc.StartContainerRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.StartContainer(ctx, req.(*grpc.StartContainerRequest), opts...)
	}
	c.handlers["grpc.StatsContainerRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.StatsContainer(ctx, req.(*grpc.StatsContainerRequest), opts...)
	}
	c.handlers["grpc.TtyWinResizeRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.TtyWinResize(ctx, req.(*grpc.TtyWinResizeRequest), opts...)
	}
	c.handlers["grpc.UpdateInterfaceRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.UpdateInterface(ctx, req.(*grpc.UpdateInterfaceRequest), opts...)
	}
	c.handlers["grpc.UpdateRoutesRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.UpdateRoutes(ctx, req.(*grpc.UpdateRoutesRequest), opts...)
	}
	c.handlers["grpc.UpdateContainerRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.UpdateContainer(ctx, req.(*grpc.UpdateContainerRequest), opts...)
	}
	c.handlers["grpc.Version"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.Version(ctx, req.(*grpc.CheckRequest), opts...)
	}
	c.handlers["grpc.WaitProcessRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.WaitProcess(ctx, req.(*grpc.WaitProcessRequest), opts...)
	}
	c.handlers["grpc.WriteStreamRequest"] = func(ctx context.Context, req interface{}, opts ...golangGrpc.CallOption) (interface{}, error) {
		return c.conn.WriteStdin(ctx, req.(*grpc.WriteStreamRequest), opts...)
	}
}

// sendReq is the main entry point: it connect to the agent, sends the
// requested gRPC message and then disconnects.
func (c *kataClient) sendReq(request interface{}) (interface{}, error) {
	if err := c.connect(); err != nil {
		return nil, err
	}

	if !c.keepAlive {
		defer c.disconnect()
	}

	msgName := proto.MessageName(request.(proto.Message))
	if msgName == "" {
		return nil, fmt.Errorf("invalid request type: %v", request)
	}

	handler := c.handlers[msgName]
	if handler == nil {
		return nil, fmt.Errorf("no handler for request %v", msgName)
	}

	message := request.(proto.Message)

	c.logger().WithField("name", msgName).WithField("request", message.String()).Debug("sending request")

	return handler(c.ctx, request)
}

func (c *kataClient) getGuestDetails(req *grpc.GuestDetailsRequest) (*grpc.GuestDetailsResponse, error) {
	c.logger().Infof("DEBUG: getGuestDetails: req: %+v", req)

	resp, err := c.sendReq(req)
	c.logger().Infof("DEBUG: getGuestDetails: resp: %+v, err: %v", resp, err)
	if err != nil {
		return nil, err
	}

	return resp.(*grpc.GuestDetailsResponse), nil
}

func (c *kataClient) getCheck(req *grpc.CheckRequest) (*grpc.HealthCheckResponse, error) {
	resp, err := c.sendReq(req)
	if err != nil {
		return nil, err
	}

	return resp.(*grpc.HealthCheckResponse), nil
}

// FIXME:
// func (c *kataClient) getVersion(req *grpc.CheckRequest) (*grpc.VersionCheckResponse, error) {
func (c *kataClient) getVersion(req *grpc.CheckRequest) (*grpc.HealthCheckResponse, error) {
	resp, err := c.sendReq(req)
	if err != nil {
		return nil, err
	}

	// FIXME:
	//_, ok := resp.(*grpc.VersionCheckResponse)
	//fmt.Printf("getVersion: grpc.VersionCheckResponse response? %b\n", ok)

	//_, ok = resp.(*grpc.HealthCheckResponse)
	//fmt.Printf("getVersion: grpc.HealthCheckResponse response? %b\n", ok)

	//return resp.(*grpc.VersionCheckResponse), nil

	return resp.(*grpc.HealthCheckResponse), nil
}

func (c *kataClient) createContainer(req *grpc.CreateContainerRequest) error {
	_, err := c.sendReq(req)

	return err
}
