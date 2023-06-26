$ErrorActionPreference = 'Stop'
Get-Content $args[0] | Select-String '^[^#[:space:]].*?:.*?##\s.*$$' | ForEach-Object { $_.ToString().Trim() } | ForEach-Object { $_.Substring(0, $_.IndexOf(':')).PadRight(30) + $_.Substring($_.IndexOf(':') + 1).Trim() }
