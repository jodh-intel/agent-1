package main

// FIXME:
//
// - build: add check to makefile that ensures all agent.proto functions are
//   defined in cli/client.go!
// - add interactive command execution?
// - work out wtf is going on with Version() type being wrong!?!
// - simplify handlers code.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kata-containers/agent/protocols/grpc"
	"github.com/sirupsen/logrus"
)

const (
	name = "kata-agent-ctl"
)

var (
	// set by the build
	version = "unknown"

	debug = false

	logger *logrus.Entry

	grpcTimeoutSecs = 30
)

func init() {
	logger = logrus.WithFields(logrus.Fields{
		"name": name,
		"pid":  os.Getpid(),
	})

	logger.Level = logrus.DebugLevel

	logger.Logger.Formatter = &logrus.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	}
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(),
		"Usage: %s [options]\n\n"+
			"Options:\n\n", os.Args[0])
	flag.PrintDefaults()

	socketPath := "unix:///tmp/kata-agent.sock"

	fmt.Fprintf(flag.CommandLine.Output(),
		"\n"+
			"Example:\n"+
			"\n"+
			"  # Start the agent\n"+
			"  $ kata-agent --debug --channelPath %q --no-udev\n"+
			"\n"+
			"  # Start the control program in another terminal\n"+
			"  $ %s --debug --agentAddress %q --keep-alive --enable-yamux\n"+
			"\n",
		socketPath, name, socketPath)
}

type junkRequest struct {
	s string
}

func (j *junkRequest) Reset() {
}

func (j *junkRequest) String() string {
	return j.s
}

func (j *junkRequest) ProtoMessage() {
}

func callSafeRoutines(client *kataClient) error {

	//------------------------------
	// guest details

	guestDetailsReq := &grpc.GuestDetailsRequest{
		MemBlockSize: true,
	}

	logger.Infof("DEBUG: calling client.getGuestDetails")
	guestDetailsResp, err := client.getGuestDetails(guestDetailsReq)
	logger.Infof("DEBUG: called client.getGuestDetails")
	if err != nil {
		return err
	}

	logger.Infof("details: %+v", guestDetailsResp)

	//------------------------------
	// junk request

	junkReq := &junkRequest{
		s: "foo",
	}

	logger.Infof("DEBUG: calling junk request")
	junkResult, err := client.sendReq(junkReq)
	logger.Infof("DEBUG: called junk request")
	logger.Infof("garbage received result: %v (err: %v)", junkResult, err)

	//------------------------------
	// check server

	checkReq := &grpc.CheckRequest{
		// Value not used
		Service: "",
	}

	checkResp, err := client.getCheck(checkReq)
	if err != nil {
		return err
	}

	logger.Infof("check: %+v", checkResp)

	//------------------------------
	// server version details

	// Note the param is the same as for getCheck()
	versionResp, err := client.getVersion(checkReq)
	if err != nil {
		return err
	}

	logger.Infof("version: %+v", versionResp)

	return nil
}

func makeRequests(client *kataClient) error {
	guestDetailsReq := &grpc.GuestDetailsRequest{
		MemBlockSize: true,
	}

	logger.Infof("DEBUG: calling client.getGuestDetails")
	guestDetailsResp, err := client.getGuestDetails(guestDetailsReq)
	logger.Infof("DEBUG: called client.getGuestDetails")

	logger.Infof("details: %+v", guestDetailsResp)

	//------------------------------

	/*
		for i := 0; i < 1024; i++ {
			err := callSafeRoutines(client)
			if err != nil {
				return err
			}
		}
	*/

	//------------------------------
	// XXX: destroy sandbox (aka server shutdown!)

	destroySandboxReq := &grpc.DestroySandboxRequest{}
	_, err = client.sendReq(destroySandboxReq)
	if err != nil {
		return err
	}

	//------------------------------
	// create sandbox

	/*
		hostname := "hostname"
		sid := "sid"
		cid := "cid"
		execID := cid
		rootPath := "/tmp/cid"

		createSandboxReq := &grpc.CreateSandboxRequest{
			Hostname:  hostname,
			SandboxId: sid,
		}

		_, err = client.sendReq(createSandboxReq)
		if err != nil {
			return err
		}

		//------------------------------
		// create container

		ociSpec := &specs.Spec{
			Version: "1.0.0-rc1",
			Process: &specs.Process{
				Args:     []string{"sh"},
				Terminal: true,
				Cwd:      rootPath,
				User: specs.User{
					UID: 0,
					GID: 0,
				},
			},
			Linux: &specs.Linux{},
			Root: &specs.Root{
				Path: rootPath,
			},
			Hostname: cid,
		}

		grpcSpec, err := grpc.OCItoGRPC(ociSpec)
		if err != nil {
			return err
		}

		createContainerReq := &grpc.CreateContainerRequest{
			ContainerId: cid,
			ExecId:      execID,
			OCI:         grpcSpec,
		}

		err = client.createContainer(createContainerReq)
		if err != nil {
			return err
		}
	*/

	//------------------------------

	return nil
}

func realMain() error {
	var showVersion bool
	var keepAlive bool
	var enableYamux bool
	var agentAddress string

	flag.BoolVar(&debug, "debug", false, "enable debug output")
	flag.BoolVar(&keepAlive, "keep-alive", false, "use a single agent connection")
	flag.BoolVar(&keepAlive, "enable-yamux", false, "use Yamux comms")
	flag.BoolVar(&showVersion, "version", false, "display program version and exit")
	flag.IntVar(&grpcTimeoutSecs, "timeout", grpcTimeoutSecs, "set gRPC timeout (seconds)")
	flag.StringVar(&agentAddress, "agentAddress", "", "address to connect to agent")

	flag.Usage = usage

	flag.Parse()

	if showVersion {
		fmt.Printf("%v version %v\n", name, version)
		return nil
	}

	ctx := context.Background()
	client, err := newKataClient(ctx, logger, agentAddress, keepAlive, enableYamux)
	if err != nil {
		return err
	}

	return makeRequests(client)
}

func main() {
	err := realMain()
	if err != nil {
		logger.WithError(err).Error("Failed")
		os.Exit(1)
	}
}
