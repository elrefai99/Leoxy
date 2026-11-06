import { execFile } from "node:child_process"
import { appendFile, mkdir, readFile, rename } from "node:fs/promises"
import os from "node:os"
import path from "node:path"
import { fileURLToPath } from "node:url"
import { promisify } from "node:util"

const execFileAsync = promisify(execFile)

const here = path.dirname(fileURLToPath(import.meta.url))
const header = "Timestamp,PID,ProcessName,WorkingSetMB,PrivateMemoryMB,CpuPercent,CpuCores"

const config = {
     processName: process.argv[2] ?? "leoxy",
     intervalMs: 1000,
     output: path.resolve(process.argv[3] ?? path.join(here, "./reports/ramMS.csv"))
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
     const current = await readFile(config.output, "utf-8").catch(() => "")
     if (current.startsWith(`${header}\n`)) return

     if (current.length > 0) {
          await rename(config.output, config.output.replace(/\.csv$/, `.${Date.now()}.csv`))
     }

     await appendFile(config.output, `${header}\n`, "utf-8")
}

function toRow(item, timestamp) {
     const workingMB = (Number(item.WorkingSet64) / 1024 ** 2).toFixed(2)
     const privateMemoryMB = (Number(item.PrivateMemorySize64) / 1024 ** 2).toFixed(2)
     const cpu = cpuUsage(item)
     const cpuPercent = cpu ? cpu.percent.toFixed(2) : ""
     const cpuCores = cpu ? cpu.cores.toFixed(2) : ""

     console.log(`Time: ${timestamp}, PID:${item.Id}, RAM:${workingMB} MB, Private:${privateMemoryMB} MB, CPU:${cpuPercent || "n/a"}% (${cpuCores || "n/a"} cores)`)

     return [
          timestamp,
          item.Id,
          item.ProcessName,
          workingMB,
          privateMemoryMB,
          cpuPercent,
          cpuCores
     ].join(",")
}

async function main() {
     await mkdir(path.dirname(config.output), { recursive: true })
     await writeHeader()

     console.log(`Monitoring process: ${config.processName}`)
     console.log(`Logical cores: ${logicalCores}`)
     console.log(`Output: ${config.output}`)
     console.log(`Press Ctrl+C to stop`)

     while (!stopped) {
          const timestamp = new Date().toISOString()
          let processes = []

          try {
               processes = await getProcess()
          }
          catch (err) {
               console.error("Failed to collect RAM/CPU metrics: ", err.message)
          }

          if (processes.length === 0) {
               console.log(`${timestamp} Process not found ${config.processName}`)
          } else {
               const rows = processes.map((item) => toRow(item, timestamp))

               try {
                    await appendFile(config.output, `${rows.join("\n")}\n`, "utf-8")
               }
               catch (err) {
                    console.error(`Failed to write ${config.output}: `, err.message)
               }
          }

          await new Promise((resolve) => setTimeout(resolve, config.intervalMs))
     }
}

main().catch(err => {
     console.error(err)
     process.exitCode = 1
})
