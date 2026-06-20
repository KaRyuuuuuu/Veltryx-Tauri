$ports = @(8080, 1420, 1421, 1422, 1423)
$stopped = @{}

foreach ($port in $ports) {
  $connections = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue
  foreach ($connection in $connections) {
    $processId = $connection.OwningProcess
    if (-not $stopped.ContainsKey($processId)) {
      try {
        Stop-Process -Id $processId -Force -ErrorAction Stop
        $stopped[$processId] = $true
        Write-Host "Stopped process $processId on port $port"
      } catch {
        Write-Host "Failed to stop process $processId on port $port"
      }
    }
  }
}

if ($stopped.Count -eq 0) {
  Write-Host "No dev process found on ports: $($ports -join ', ')"
}
