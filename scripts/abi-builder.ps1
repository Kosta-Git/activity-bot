foreach ($file in Get-ChildItem ./abi/*.json) {
  $pkg = $file.BaseName
  $pkg_name = $pkg.Substring(0,1).ToLower() + $pkg.Substring(1)
  $pkg_dir = "./pkg/abi/$pkg_name"
  $outFile = "$pkg_dir/$pkg_name.go"

  # Create package directory if it does not exist
  if (-not (Test-Path -Path $pkg_dir -PathType Container)) {
    New-Item -ItemType Directory -Path $pkg_dir | Out-Null
  }

  Write-Host "Processing $file ..."
  abigen --abi $file --pkg $pkg_name --out $outFile
}