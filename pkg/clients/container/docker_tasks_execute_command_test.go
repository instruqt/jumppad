package container

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/instruqt/jumppad/pkg/clients/container/mocks"
	imocks "github.com/instruqt/jumppad/pkg/clients/images/mocks"
	"github.com/instruqt/jumppad/pkg/clients/logger"
	"github.com/instruqt/jumppad/pkg/clients/tar"
	"github.com/instruqt/jumppad/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testExecCommandSetup(t *testing.T) (*DockerTasks, *mocks.Docker, *imocks.ImageLog) {
	// we need to add the stream index (stdout) as the first byte for the hijacker
	writerOutput := []byte("log output")
	writerOutput = append([]byte{1}, writerOutput...)

	mk := &mocks.Docker{}
	mk.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	mk.On("Info", mock.Anything).Return(system.Info{Driver: StorageDriverOverlay2}, nil)
	mk.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).Return(container.ExecCreateResponse{ID: "abc"}, nil)
	mk.On("ContainerExecAttach", mock.Anything, mock.Anything, mock.Anything).Return(
		types.HijackedResponse{
			Conn: &net.TCPConn{},
			Reader: bufio.NewReader(
				bytes.NewReader(writerOutput),
			),
		},
		nil,
	)
	mk.On("ContainerExecStart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mk.On("ContainerExecInspect", mock.Anything, mock.Anything, mock.Anything).Return(container.ExecInspect{Running: false, ExitCode: 0}, nil)

	il := &imocks.ImageLog{}

	dt, _ := NewDockerTasks(mk, il, &tar.TarGz{}, logger.NewTestLogger(t))
	dt.defaultWait = 1 * time.Millisecond
	return dt, mk, il
}

func TestExecuteCommandCreatesExec(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	dt, mk, _ := testExecCommandSetup(t)
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := dt.ExecuteCommand("testcontainer", command, []string{"abc=123"}, "/files", "1000", "2000", 300, writer)
	assert.NoError(t, err)

	mk.AssertCalled(t, "ContainerExecCreate", mock.Anything, "testcontainer", mock.Anything)
	params := testutils.GetCalls(&mk.Mock, "ContainerExecCreate")[0].Arguments[2].(container.ExecOptions)

	// test the command
	assert.Equal(t, params.Cmd[0], command[0])

	// test the working directory
	assert.Equal(t, params.WorkingDir, "/files")

	// check the environment variables
	assert.Equal(t, params.Env[0], "abc=123")

	// check the user
	assert.Equal(t, params.User, "1000:2000")
}

func TestExecuteCommandExecFailReturnError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	md, mk, _ := testExecCommandSetup(t)
	testutils.RemoveOn(&mk.Mock, "ContainerExecCreate")
	mk.On("ContainerExecCreate", mock.Anything, mock.Anything, mock.Anything).Return(container.ExecCreateResponse{}, fmt.Errorf("boom"))

	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", 300, writer)
	assert.Error(t, err)
}

func TestExecuteCommandAttachesToExec(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	md, mk, _ := testExecCommandSetup(t)
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", 300, writer)
	assert.NoError(t, err)

	mk.AssertCalled(t, "ContainerExecAttach", mock.Anything, "abc", mock.Anything)
}

func TestExecuteCommandAttachFailReturnError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	md, mk, _ := testExecCommandSetup(t)
	testutils.RemoveOn(&mk.Mock, "ContainerExecAttach")
	mk.On("ContainerExecAttach", mock.Anything, "abc", mock.Anything).Return(types.HijackedResponse{}, fmt.Errorf("boom"))
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", 300, writer)
	assert.Error(t, err)
}

//func TestExecuteCommandStartsExec(t *testing.T) {
//	if testing.Short() {
//		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
//	}
//
//	mk, mic := testExecCommandMockSetup()
//	md := NewDockerTasks(mk, mic, &TarGz{}, clients.NewTestLogger(t))
//	writer := bytes.NewBufferString("")
//
//	command := []string{"ls", "-las"}
//	err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", writer)
//	assert.NoError(t, err)
//
//	mk.AssertCalled(t, "ContainerExecStart", mock.Anything, "abc", mock.Anything)
//}
//
//func TestExecuteStartsFailReturnsError(t *testing.T) {
//	if testing.Short() {
//		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
//	}
//
//	mk, mic := testExecCommandMockSetup()
//	testutils.RemoveOn(&mk.Mock, "ContainerExecStart")
//	mk.On("ContainerExecStart", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))
//	md := NewDockerTasks(mk, mic, &TarGz{}, clients.NewTestLogger(t))
//	writer := bytes.NewBufferString("")
//
//	command := []string{"ls", "-las"}
//	err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", writer)
//	assert.Error(t, err)
//}

func TestExecuteCommandInspectsExecAndReturnsErrorOnFail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test on Github actions as this test times out for an unknown reason, can't diagnose the problem")
	}

	md, mk, _ := testExecCommandSetup(t)
	testutils.RemoveOn(&mk.Mock, "ContainerExecInspect")
	mk.On("ContainerExecInspect", mock.Anything, mock.Anything, mock.Anything).Return(container.ExecInspect{Running: false, ExitCode: 1}, nil)
	writer := bytes.NewBufferString("")

	command := []string{"ls", "-las"}
	_, err := md.ExecuteCommand("testcontainer", command, nil, "/", "", "", 300, writer)
	assert.Error(t, err)

	mk.AssertCalled(t, "ContainerExecInspect", mock.Anything, "abc", mock.Anything)
}
