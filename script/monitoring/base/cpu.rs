use std::collections::HashMap;
use std::process::Command;
use std::thread;
use std::time::{Duration, Instant};

fn main() {
    let cores = thread::available_parallelism()
        .map(|count| count.get())
        .unwrap_or(1) as f64;
    let mut previous: HashMap<u32, (f64, Instant)> = HashMap::new();

    println!("Monitoring Leoxy CPU usage. Press Ctrl+C to stop.");

    loop {
        let sampled_at = Instant::now();
        let output = Command::new("powershell.exe")
            .args([
                "-NoProfile",
                "-NonInteractive",
                "-Command",
                "Get-Process -Name leoxy -ErrorAction SilentlyContinue | ForEach-Object { \"$($_.Id),$($_.ProcessName),$($_.TotalProcessorTime.TotalSeconds)\" }",
            ])
            .output();

        match output {
            Ok(result) if result.status.success() => {
                let processes = String::from_utf8_lossy(&result.stdout);
                if processes.trim().is_empty() {
                    println!("Leoxy process not found");
                } else {
                    for line in processes.lines() {
                        let fields: Vec<&str> = line.trim().split(',').collect();
                        if fields.len() != 3 {
                            continue;
                        }

                        let Ok(pid) = fields[0].parse::<u32>() else {
                            continue;
                        };
                        let Ok(cpu_seconds) = fields[2].parse::<f64>() else {
                            continue;
                        };

                        if let Some((old_cpu_seconds, old_sampled_at)) =
                            previous.insert(pid, (cpu_seconds, sampled_at))
                        {
                            let elapsed = sampled_at.duration_since(old_sampled_at).as_secs_f64();
                            if elapsed > 0.0 {
                                let used_cores = (cpu_seconds - old_cpu_seconds) / elapsed;
                                let percent = used_cores / cores * 100.0;
                                println!(
                                    "PID: {pid}, Process: {}, CPU: {percent:.2}% ({used_cores:.2} cores)",
                                    fields[1]
                                );
                            }
                        } else {
                            println!("PID: {pid}, Process: {}, CPU: sampling...", fields[1]);
                        }
                    }
                }
            }
            Ok(result) => eprintln!(
                "Failed to read Leoxy CPU: {}",
                String::from_utf8_lossy(&result.stderr).trim()
            ),
            Err(error) => eprintln!("Failed to run PowerShell: {error}"),
        }

        thread::sleep(Duration::from_secs(1));
    }
}
