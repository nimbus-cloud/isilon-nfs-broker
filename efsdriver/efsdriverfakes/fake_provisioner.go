// This file was generated by counterfeiter
package efsdriverfakes

import (
	"sync"

	"code.cloudfoundry.org/efsbroker/efsdriver"
	"github.com/aws/aws-sdk-go/service/efs"
)

type FakeEFSProvisioner struct {
	CreateFileSystemStub        func(*efs.CreateFileSystemInput) (*efs.FileSystemDescription, error)
	createFileSystemMutex       sync.RWMutex
	createFileSystemArgsForCall []struct {
		arg1 *efs.CreateFileSystemInput
	}
	createFileSystemReturns struct {
		result1 *efs.FileSystemDescription
		result2 error
	}
	DeleteFileSystemStub        func(*efs.DeleteFileSystemInput) (*efs.DeleteFileSystemOutput, error)
	deleteFileSystemMutex       sync.RWMutex
	deleteFileSystemArgsForCall []struct {
		arg1 *efs.DeleteFileSystemInput
	}
	deleteFileSystemReturns struct {
		result1 *efs.DeleteFileSystemOutput
		result2 error
	}
}

func (fake *FakeEFSProvisioner) CreateFileSystem(arg1 *efs.CreateFileSystemInput) (*efs.FileSystemDescription, error) {
	fake.createFileSystemMutex.Lock()
	fake.createFileSystemArgsForCall = append(fake.createFileSystemArgsForCall, struct {
		arg1 *efs.CreateFileSystemInput
	}{arg1})
	fake.createFileSystemMutex.Unlock()
	if fake.CreateFileSystemStub != nil {
		return fake.CreateFileSystemStub(arg1)
	} else {
		return fake.createFileSystemReturns.result1, fake.createFileSystemReturns.result2
	}
}

func (fake *FakeEFSProvisioner) CreateFileSystemCallCount() int {
	fake.createFileSystemMutex.RLock()
	defer fake.createFileSystemMutex.RUnlock()
	return len(fake.createFileSystemArgsForCall)
}

func (fake *FakeEFSProvisioner) CreateFileSystemArgsForCall(i int) *efs.CreateFileSystemInput {
	fake.createFileSystemMutex.RLock()
	defer fake.createFileSystemMutex.RUnlock()
	return fake.createFileSystemArgsForCall[i].arg1
}

func (fake *FakeEFSProvisioner) CreateFileSystemReturns(result1 *efs.FileSystemDescription, result2 error) {
	fake.CreateFileSystemStub = nil
	fake.createFileSystemReturns = struct {
		result1 *efs.FileSystemDescription
		result2 error
	}{result1, result2}
}

func (fake *FakeEFSProvisioner) DeleteFileSystem(arg1 *efs.DeleteFileSystemInput) (*efs.DeleteFileSystemOutput, error) {
	fake.deleteFileSystemMutex.Lock()
	fake.deleteFileSystemArgsForCall = append(fake.deleteFileSystemArgsForCall, struct {
		arg1 *efs.DeleteFileSystemInput
	}{arg1})
	fake.deleteFileSystemMutex.Unlock()
	if fake.DeleteFileSystemStub != nil {
		return fake.DeleteFileSystemStub(arg1)
	} else {
		return fake.deleteFileSystemReturns.result1, fake.deleteFileSystemReturns.result2
	}
}

func (fake *FakeEFSProvisioner) DeleteFileSystemCallCount() int {
	fake.deleteFileSystemMutex.RLock()
	defer fake.deleteFileSystemMutex.RUnlock()
	return len(fake.deleteFileSystemArgsForCall)
}

func (fake *FakeEFSProvisioner) DeleteFileSystemArgsForCall(i int) *efs.DeleteFileSystemInput {
	fake.deleteFileSystemMutex.RLock()
	defer fake.deleteFileSystemMutex.RUnlock()
	return fake.deleteFileSystemArgsForCall[i].arg1
}

func (fake *FakeEFSProvisioner) DeleteFileSystemReturns(result1 *efs.DeleteFileSystemOutput, result2 error) {
	fake.DeleteFileSystemStub = nil
	fake.deleteFileSystemReturns = struct {
		result1 *efs.DeleteFileSystemOutput
		result2 error
	}{result1, result2}
}

var _ efsdriver.EFSProvisioner = new(FakeEFSProvisioner)
