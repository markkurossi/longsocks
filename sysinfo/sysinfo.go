//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package sysinfo

import (
	"context"
	"runtime"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

type HostInfo struct {
	OS                 string `json:"os"`
	Distribution       string `json:"distribution"`
	DistributionFamily string `json:"distribution_family"`
	Version            string `json:"version"`
	Arch               string `json:"arch"`
	KernelVersion      string `json:"kernel_version"`
	KernelArch         string `json:"kernel_arch"`
}

type AgentInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

type MemInfo struct {
	Total     uint64 `json:"total"`
	Available uint64 `json:"available"`
}

type SwapInfo struct {
	Total uint64 `json:"total"`
	Free  uint64 `json:"free"`
}

func Host(ctx context.Context) (*HostInfo, error) {
	hi, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, err
	}

	return &HostInfo{
		OS:                 hi.OS,
		Distribution:       hi.Platform,
		Version:            hi.PlatformVersion,
		Arch:               runtime.GOARCH,
		KernelVersion:      hi.KernelVersion,
		KernelArch:         hi.KernelArch,
		DistributionFamily: hi.PlatformFamily,
	}, nil
}

func Agent(ctx context.Context) (*AgentInfo, error) {
	return &AgentInfo{
		Version:   "0.0.1", // Longsocks diald version
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}, nil
}

func Memory(ctx context.Context) (*MemInfo, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return &MemInfo{
		Total:     vm.Total,
		Available: vm.Available,
	}, nil
}

func Swap(ctx context.Context) (*SwapInfo, error) {
	swap, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return &SwapInfo{
		Total: swap.Total,
		Free:  swap.Free,
	}, nil
}
