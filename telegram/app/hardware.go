package app

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/yangrq1018/jerry-bot/telegram"
	"github.com/yangrq1018/jerry-bot/util"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

func ServerStatsCommand() telegram.Command {
	return SimpleCommand{
		name:        "server",
		description: "查询服务器资源状态",
		handle: func(b *telegram.Bot, u tgbotapi.Update) error {
			buf := bytes.NewBuffer(nil)
			SnapshotHardware(buf)
			b.ReplyTo(*u.Message, buf.String())
			return nil
		},
	}
}

type CPUStats struct {
	coreCountPhysical int
	coreCountLogical  int
	totalUsage        float64
	usagePerCore      []float64
}

type MemStats struct {
	total     uint64
	available uint64
	used      uint64
}

type Kernel struct {
	platform string
	family   string
	version  string
}

type hardwareStats struct {
	// cpu
	cpu CPUStats

	// Physical memory
	mem MemStats

	// Top memory consumers
	topMemConsumer []memConsumer

	// Boot time
	bootTime time.Time

	// Kernel info
	kernel Kernel
}

type memConsumer struct {
	process string
	data    float32
}

const (
	GBToB uint64 = 1024 * 1024 * 1024
)

func bytesToGigabytes(b uint64) float64 {
	return float64(b) / float64(GBToB)
}

func (h hardwareStats) Write(w io.Writer) {
	fmt.Fprintf(w, "---CPU---\n")
	fmt.Fprintf(w, "Total: %d\n", h.cpu.coreCountPhysical)
	fmt.Fprintf(w, "Total (logical): %d\n", h.cpu.coreCountLogical)
	fmt.Fprintf(w, "Usage: %.2f%%\n", h.cpu.totalUsage)
	fmt.Fprintf(w, "Usage (per core):\n")
	for i, u := range h.cpu.usagePerCore {
		fmt.Fprintf(w, "  Core %d: %.2f%%\n", i+1, u)
	}

	fmt.Fprintf(w, "---Memory---\n")
	fmt.Fprintf(w, "%s: %.2f GB\n", "Total", bytesToGigabytes(h.mem.total)) // as GB
	fmt.Fprintf(w, "%s: %.2f GB\n", "Available", bytesToGigabytes(h.mem.available))
	fmt.Fprintf(w, "%s: %.2f GB\n", "Used", bytesToGigabytes(h.mem.used))

	fmt.Fprintf(w, "---Top 5 memory process---\n")
	for i, p := range h.topMemConsumer {
		fmt.Fprintf(w, "#%d[%s]:%.2f%%\n", i+1, p.process, p.data)
	}

	fmt.Fprintf(w, "---Boot time---\n")
	fmt.Fprintf(w, "last boot time: %s\n", h.bootTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "have kept running for %d days\n", int(time.Now().Sub(h.bootTime).Hours()/24))

	fmt.Fprintf(w, "---Kernel---\n")
	fmt.Fprintf(w, "platform: %s\n", h.kernel.platform)
	fmt.Fprintf(w, "family: %s\n", h.kernel.family)
	fmt.Fprintf(w, "version: %s\n", h.kernel.version)
}

func SnapshotHardware(w io.Writer) {
	physicalCount, _ := cpu.Counts(false)
	logicalCount, _ := cpu.Counts(true)
	totalCPUUsage, _ := cpu.Percent(3*time.Second, false)
	totalCPUUsagePerCore, _ := cpu.Percent(3*time.Second, true)

	ms, _ := mem.VirtualMemory()

	var hs hardwareStats
	hs.cpu.coreCountPhysical = physicalCount
	hs.cpu.coreCountLogical = logicalCount
	hs.cpu.totalUsage = totalCPUUsage[0]
	hs.cpu.usagePerCore = totalCPUUsagePerCore

	hs.mem.total = ms.Total
	hs.mem.available = ms.Available
	hs.mem.used = ms.Used

	processes, _ := process.Processes()
	sort.Slice(processes, func(i, j int) bool {
		memI, _ := processes[i].MemoryPercent()
		memJ, _ := processes[j].MemoryPercent()
		return memI > memJ
	})

	for _, p := range processes[:util.Min(5, len(processes))] {
		name, _ := p.Name()
		mp, _ := p.MemoryPercent()
		hs.topMemConsumer = append(hs.topMemConsumer, memConsumer{
			process: name,
			data:    mp,
		})
	}
	bt, _ := host.BootTime()
	hs.bootTime = time.Unix(int64(bt), 0)

	platform, family, version, _ := host.PlatformInformation()
	hs.kernel = Kernel{
		platform: platform,
		family:   family,
		version:  version,
	}
	hs.Write(w)
}
