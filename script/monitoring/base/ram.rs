use std::process::Command;
use std::thread;
use std::time::Duration;

fn main() {
    println!("Monitoring Leoxy RAM usage. Press Ctrl+C to stop.");

    loop {
        let output = Command::new("powershell.exe")
            .args([
                "-NoProfile",
                "-NonInteractive",
                "-Command",
                "Get-Process -Name leoxy -ErrorAction SilentlyContinue | ForEach-Object { \"$($_.Id),$($_.ProcessName),$($_.WorkingSet64)\" }",
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
                        if fields.len() == 3 {
                            if let (Ok(pid), Ok(bytes)) =
                                (fields[0].parse::<u32>(), fields[2].parse::<u64>())
                            {
                                println!(
                                    "PID: {pid}, Process: {}, RAM: {:.2} MB",
                                    fields[1],
                                    bytes as f64 / 1_048_576.0
                                );
                            }
                        }
                    }
                }
            }
            Ok(result) => eprintln!(
                "Failed to read Leoxy RAM: {}",
                String::from_utf8_lossy(&result.stderr).trim()
            ),
            Err(error) => eprintln!("Failed to run PowerShell: {error}"),
        }

        thread::sleep(Duration::from_secs(1));
    }
}
