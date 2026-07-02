[CmdletBinding()]
param(
  [switch]$Fix
)

$ErrorActionPreference = 'Stop'

function Get-CodePage {
  $line = & chcp.com
  if ($line -match '(\d+)') {
    return [int]$Matches[1]
  }
  return $null
}

function Set-Utf8Encoding {
  $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
  [Console]::InputEncoding = $utf8NoBom
  [Console]::OutputEncoding = $utf8NoBom
  $global:OutputEncoding = $utf8NoBom
  $global:PSDefaultParameterValues['Get-Content:Encoding'] = 'UTF8'
  $global:PSDefaultParameterValues['Set-Content:Encoding'] = 'UTF8'
  $global:PSDefaultParameterValues['Add-Content:Encoding'] = 'UTF8'
  $global:PSDefaultParameterValues['Out-File:Encoding'] = 'UTF8'
  $global:PSDefaultParameterValues['Select-String:Encoding'] = 'UTF8'

  if (Get-Command chcp.com -ErrorAction SilentlyContinue) {
    & chcp.com 65001 > $null
  }

  if (Get-Command git -ErrorAction SilentlyContinue) {
    Set-GitConfigIfNeeded 'core.quotepath' 'false'
    Set-GitConfigIfNeeded 'i18n.logOutputEncoding' 'utf-8'
    Set-GitConfigIfNeeded 'i18n.commitEncoding' 'utf-8'
  }
}

function Set-GitConfigIfNeeded {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$Value
  )

  $current = git config --global --get $Name 2>$null
  if ($current -ne $Value) {
    git config --global $Name $Value
  }
}

if ($Fix) {
  Set-Utf8Encoding
}

$codePage = Get-CodePage
$gitQuotePath = if (Get-Command git -ErrorAction SilentlyContinue) {
  git config --global --get core.quotepath
} else {
  '<git not found>'
}
$gitLogEncoding = if (Get-Command git -ErrorAction SilentlyContinue) {
  git config --global --get i18n.logOutputEncoding
} else {
  '<git not found>'
}
$gitCommitEncoding = if (Get-Command git -ErrorAction SilentlyContinue) {
  git config --global --get i18n.commitEncoding
} else {
  '<git not found>'
}

function Test-DefaultEncoding {
  param([string]$Key)
  return $PSDefaultParameterValues.ContainsKey($Key) -and
    $PSDefaultParameterValues[$Key] -eq 'UTF8'
}

$checks = @(
  [pscustomobject]@{
    Name = 'Active code page'
    Expected = '65001'
    Actual = [string]$codePage
    Pass = $codePage -eq 65001
  },
  [pscustomobject]@{
    Name = 'Console input encoding'
    Expected = 'Unicode (UTF-8)'
    Actual = [Console]::InputEncoding.EncodingName
    Pass = [Console]::InputEncoding.WebName -eq 'utf-8'
  },
  [pscustomobject]@{
    Name = 'Console output encoding'
    Expected = 'Unicode (UTF-8)'
    Actual = [Console]::OutputEncoding.EncodingName
    Pass = [Console]::OutputEncoding.WebName -eq 'utf-8'
  },
  [pscustomobject]@{
    Name = 'PowerShell pipe output encoding'
    Expected = 'Unicode (UTF-8)'
    Actual = $OutputEncoding.EncodingName
    Pass = $OutputEncoding.WebName -eq 'utf-8'
  },
  [pscustomobject]@{
    Name = 'Git quoted paths'
    Expected = 'false'
    Actual = [string]$gitQuotePath
    Pass = $gitQuotePath -eq 'false'
  },
  [pscustomobject]@{
    Name = 'Git log output encoding'
    Expected = 'utf-8'
    Actual = [string]$gitLogEncoding
    Pass = $gitLogEncoding -eq 'utf-8'
  },
  [pscustomobject]@{
    Name = 'Git commit encoding'
    Expected = 'utf-8'
    Actual = [string]$gitCommitEncoding
    Pass = $gitCommitEncoding -eq 'utf-8'
  },
  [pscustomobject]@{
    Name = 'Get-Content default encoding'
    Expected = 'UTF8'
    Actual = [string]$PSDefaultParameterValues['Get-Content:Encoding']
    Pass = Test-DefaultEncoding 'Get-Content:Encoding'
  },
  [pscustomobject]@{
    Name = 'Select-String default encoding'
    Expected = 'UTF8'
    Actual = [string]$PSDefaultParameterValues['Select-String:Encoding']
    Pass = Test-DefaultEncoding 'Select-String:Encoding'
  },
  [pscustomobject]@{
    Name = 'Out-File default encoding'
    Expected = 'UTF8'
    Actual = [string]$PSDefaultParameterValues['Out-File:Encoding']
    Pass = Test-DefaultEncoding 'Out-File:Encoding'
  }
)

$checks | Format-Table -AutoSize

if ($checks.Pass -contains $false) {
  Write-Error 'PowerShell UTF-8 encoding checks failed. Run in the affected PowerShell session: .\scripts\check_powershell_encoding.ps1 -Fix'
}
