package main

import (
	"syscall"

	gpb "github.com/gogo/protobuf/types"
	"github.com/kata-containers/agent/pkg/types"
	pb "github.com/kata-containers/agent/protocols/grpc"
	"golang.org/x/net/context"
)

type nopAgentGRPC struct {
	sandbox *sandbox
	version string
}

func (n *nopAgentGRPC) CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) StartContainer(ctx context.Context, req *pb.StartContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) RemoveContainer(ctx context.Context, req *pb.RemoveContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) ExecProcess(ctx context.Context, req *pb.ExecProcessRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) SignalProcess(ctx context.Context, req *pb.SignalProcessRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) WaitProcess(ctx context.Context, req *pb.WaitProcessRequest) (*pb.WaitProcessResponse, error) {
	return &pb.WaitProcessResponse{}, nil
}

func (n *nopAgentGRPC) ListProcesses(ctx context.Context, req *pb.ListProcessesRequest) (*pb.ListProcessesResponse, error) {
	return &pb.ListProcessesResponse{}, nil
}

func (n *nopAgentGRPC) UpdateContainer(ctx context.Context, req *pb.UpdateContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) StatsContainer(ctx context.Context, req *pb.StatsContainerRequest) (*pb.StatsContainerResponse, error) {
	return &pb.StatsContainerResponse{}, nil
}

func (n *nopAgentGRPC) PauseContainer(ctx context.Context, req *pb.PauseContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) ResumeContainer(ctx context.Context, req *pb.ResumeContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) WriteStdin(ctx context.Context, req *pb.WriteStreamRequest) (*pb.WriteStreamResponse, error) {
	return &pb.WriteStreamResponse{}, nil
}

func (n *nopAgentGRPC) ReadStdout(ctx context.Context, req *pb.ReadStreamRequest) (*pb.ReadStreamResponse, error) {
	return &pb.ReadStreamResponse{}, nil
}

func (n *nopAgentGRPC) ReadStderr(ctx context.Context, req *pb.ReadStreamRequest) (*pb.ReadStreamResponse, error) {
	return &pb.ReadStreamResponse{}, nil
}

func (n *nopAgentGRPC) CloseStdin(ctx context.Context, req *pb.CloseStdinRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) TtyWinResize(ctx context.Context, req *pb.TtyWinResizeRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) AddInterface(ctx context.Context, req *pb.AddInterfaceRequest) (*types.Interface, error) {
	return &types.Interface{}, nil
}

func (n *nopAgentGRPC) UpdateInterface(ctx context.Context, req *pb.UpdateInterfaceRequest) (*types.Interface, error) {
	return &types.Interface{}, nil
}

func (n *nopAgentGRPC) RemoveInterface(ctx context.Context, req *pb.RemoveInterfaceRequest) (*types.Interface, error) {
	return &types.Interface{}, nil
}

func (n *nopAgentGRPC) UpdateRoutes(ctx context.Context, req *pb.UpdateRoutesRequest) (*pb.Routes, error) {
	return req.Routes, nil
}

func (n *nopAgentGRPC) ListInterfaces(ctx context.Context, req *pb.ListInterfacesRequest) (*pb.Interfaces, error) {
	return &pb.Interfaces{}, nil
}

func (n *nopAgentGRPC) ListRoutes(ctx context.Context, req *pb.ListRoutesRequest) (*pb.Routes, error) {
	return &pb.Routes{}, nil
}

func (n *nopAgentGRPC) CreateSandbox(ctx context.Context, req *pb.CreateSandboxRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) DestroySandbox(ctx context.Context, req *pb.DestroySandboxRequest) (*gpb.Empty, error) {

	agentLog.Infof("DEBUG: nopAgentGRPC.DestroySandbox: calling requestServerStop")
	n.requestServerStop()
	agentLog.Infof("DEBUG: nopAgentGRPC.DestroySandbox: called requestServerStop")

	return emptyResp, nil
}

func (n *nopAgentGRPC) requestServerStop() {
	agentLog.Infof("DEBUG: requestServerStop: ")

	// Synchronize the caches on the system. This is needed to ensure
	// there are no pending transactions left before the VM is shut down.
	syscall.Sync()

	agentLog.Infof("DEBUG: requestServerStop: sending shutdown to server")

	//// Run in a separate thread to ensure the gRPC server thread doesn't
	//// race with the main thread.
	//go func() {

	// Inform the caller the server should end
	n.sandbox.shutdown <- true

	//}()

	agentLog.Infof("DEBUG: requestServerStop: sent shutdown to server")
}

func (n *nopAgentGRPC) OnlineCPUMem(ctx context.Context, req *pb.OnlineCPUMemRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) ReseedRandomDev(ctx context.Context, req *pb.ReseedRandomDevRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) GetGuestDetails(ctx context.Context, req *pb.GuestDetailsRequest) (*pb.GuestDetailsResponse, error) {
	agentLog.Infof("FIXME: nopAgentGRPC.GetGuestDetails: req: %+v", req)

	return &pb.GuestDetailsResponse{}, nil
}

func (n *nopAgentGRPC) SetGuestDateTime(ctx context.Context, req *pb.SetGuestDateTimeRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) CopyFile(ctx context.Context, req *pb.CopyFileRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (n *nopAgentGRPC) Check(ctx context.Context, req *pb.CheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: pb.HealthCheckResponse_SERVING}, nil
}

func (n *nopAgentGRPC) Version(ctx context.Context, req *pb.CheckRequest) (*pb.VersionCheckResponse, error) {
	return &pb.VersionCheckResponse{
		GrpcVersion:  pb.APIVersion,
		AgentVersion: n.version,
	}, nil

}
