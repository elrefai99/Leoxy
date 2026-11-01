import { execFile } from "node:child_process"
import { appendFile, mkdir, stat } from "node:fs/promises"
import os from "node:os"
import path from "node:path"
import { promisify } from "node:util"

const execFileAsync = promisify(execFile)

const config = {
     processName: process.argv[2] ?? "leoxy",
     intervalMs: 1000,
     output: path.resolve("../../docs/baseline/ramMS.csv")
}
const logicalCores = os.cpus().length
const previous = new Map()
let stopped = false

function stop() {
     stopped = true
}

process.on("SIGINT", stop)
process.on("SIGTERM", stop)

async function getProcess() {
     const name = config.processName.replace(/'/g, "''")
     const script = `
$items = Get-Process -Name '${name}' -ErrorAction SilentlyContinue
if ($items) {
     $items | Select-Object Id, ProcessName, WorkingSet64, PrivateMemorySize64,
          @{n='CpuSeconds';e={$_.TotalProcessorTime.TotalSeconds}},
          @{n='NowSec';e={[Diagnostics.Stopwatch]::GetTimestamp() / [Diagnostics.Stopwatch]::Frequency}} |
          ConvertTo-Json -Compress
}
`

     const { stdout } = await execFileAsync("powershell.exe", [
          "-NoProfile",
          "-NonInteractive",
          "-Command",
          script
     ], {
          windowsHide: true,
          timeout: 5000
     })

     if (!stdout.trim()) return []

     const result = JSON.parse(stdout)

     return Array.isArray(result) ? result : [result]
}

function cpuUsage(item) {
     const cpu = Number(item.CpuSeconds)
     const now = Number(item.NowSec)
     const prev = previous.get(item.Id)
     previous.set(item.Id, { cpu, now })

     if (!prev) return null

     const wall = now - prev.now
     if (wall <= 0) return null

     const cores = (cpu - prev.cpu) / wall

     return { cores, percent: (cores / logicalCores) * 100 }
}

async function writeHeader() {
     const size = await stat(config.output).then((s) => s.size).catch(() => 0)
     if (size > 0) return

     await appendFile(
          config.output,
          "Timestamp,PID,ProcessName,WorkingSetMB,PrivateMemoryMB,CpuPercent,CpuCores\n",
          "utf-8"
     )
}

async function main() {
     await mkdir(path.dirname(config.output), { recursive: true })
     await writeHeader()

     console.log(`Monitoring process: ${config.processName}`)
     console.log(`Logical cores: ${logicalCores}`)
     console.log(`Output: ${config.output}`)
     console.log(`Press Ctrl+C to stop`)

     while (!stopped) {
          try {
               const processes = await getProcess()
               const timestamp = new Date().toISOString()

               for (const item of processes) {
                    const workingMB = (
                         Number(item.WorkingSet64) / 1024 ** 2
                    ).toFixed(2)
                    const privateMemoryMB = (
                         Number(item.PrivateMemorySize64) / 1024 ** 2
                    ).toFixed(2)
                    const cpu = cpuUsage(item)
                    const cpuPercent = cpu ? cpu.percent.toFixed(2) : ""
                    const cpuCores = cpu ? cpu.cores.toFixed(2) : ""

                    const row = [
                         timestamp,
                         item.Id,
                         item.ProcessName,
                         workingMB,
                         privateMemoryMB,
                         cpuPercent,
                         cpuCores
                    ].join(",")

                    await appendFile(config.output, `${row}\n`, "utf-8")
                    console.log(`Time: ${timestamp}, PID:${item.Id}, RAM:${workingMB} MB, Private:${privateMemoryMB} MB, CPU:${cpuPercent || "n/a"}% (${cpuCores || "n/a"} cores)`)
               }

               if (processes.length === 0) {
                    console.log(`${timestamp} Process not found ${config.processName}`)
               }
          }
          catch (err) {
               console.error("Failed to collect RAM/CPU metrics: ", err.message)
          }

          await new Promise((resolve) => setTimeout(resolve, config.intervalMs))
     }
}

main().catch(err => {
     console.error(err)
     process.exitCode = 1
})
