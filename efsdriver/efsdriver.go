package efsdriver

import "github.com/aws/aws-sdk-go/service/efs"

//go:generate counterfeiter -o efsdriverfakes/fake_provisioner.go . EFSProvisioner

type EFSProvisioner interface {
	CreateFileSystem(*efs.CreateFileSystemInput) (*efs.FileSystemDescription, error)

	DeleteFileSystem(*efs.DeleteFileSystemInput) (*efs.DeleteFileSystemOutput, error)
}
