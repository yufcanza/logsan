param(
    [string]$RepoRoot = (Get-Location).Path,
    [string]$OutRoot = ''
)


# Embedded common helpers. This file is standalone and can be run from the repository root.

Set-StrictMode -Version 2.0

function Get-CheckGoCommand {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) {
        return $go.Source
    }

    throw 'go executable was not found in PATH. Install Go and make sure go is available in PATH.'
}

function New-CheckContext {
    param(
        [Parameter(Mandatory=$true)][string]$Student,
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [string]$OutRoot = ''
    )

    $repo = (Resolve-Path -LiteralPath $RepoRoot).Path
    if ($OutRoot -eq '') {
        $OutRoot = Join-Path $repo '.check-results'
    }

    $timestamp = Get-Date -Format 'yyyyMMdd_HHmmss'
    $safeStudent = $Student -replace '[^A-Za-z0-9_.-]', '_'
    $resultDir = Join-Path $OutRoot "${safeStudent}_${timestamp}"
    $logsDir = Join-Path $resultDir 'logs'
    $inputsDir = Join-Path $resultDir 'inputs'
    $outputsDir = Join-Path $resultDir 'outputs'
    $metaDir = Join-Path $resultDir 'meta'
    $tmpDir = Join-Path $resultDir 'tmp'

    foreach ($dir in @($resultDir, $logsDir, $inputsDir, $outputsDir, $metaDir, $tmpDir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }

    $ctx = [ordered]@{
        Student = $Student
        RepoRoot = $repo
        ResultDir = $resultDir
        LogsDir = $logsDir
        InputsDir = $inputsDir
        OutputsDir = $outputsDir
        MetaDir = $metaDir
        TmpDir = $tmpDir
        CommandsPath = Join-Path $resultDir 'commands.jsonl'
        GoCmd = Get-CheckGoCommand
        StartedAt = (Get-Date).ToString('o')
        CommandResults = @{}
        Assessments = New-Object System.Collections.ArrayList
    }

    '' | Set-Content -LiteralPath $ctx.CommandsPath -Encoding UTF8
    return $ctx
}

function Write-CheckText {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$RelativePath,
        [Parameter(Mandatory=$true)][string]$Content
    )

    $path = Join-Path $Ctx.ResultDir $RelativePath
    $parent = Split-Path -Parent $path
    if ($parent) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
    Set-Content -LiteralPath $path -Value $Content -Encoding UTF8
    return $path
}

function Save-CheckJson {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)]$Value
    )

    $json = $Value | ConvertTo-Json -Depth 30
    Set-Content -LiteralPath $Path -Value $json -Encoding UTF8
}

function Invoke-CheckCommand {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Name,
        [Parameter(Mandatory=$true)][string]$Command,
        [string]$WorkingDirectory = ''
    )

    if ($WorkingDirectory -eq '') {
        $WorkingDirectory = $Ctx.RepoRoot
    }

    $safeName = $Name -replace '[^A-Za-z0-9_.-]', '_'
    $logPath = Join-Path $Ctx.LogsDir "$safeName.log"
    $runnerPath = Join-Path $Ctx.TmpDir "$safeName.ps1"
    $started = Get-Date

    $runner = @"
`$ErrorActionPreference = 'Continue'
Set-Location -LiteralPath '$($WorkingDirectory.Replace("'", "''"))'
$Command
`$exitCode = `$global:LASTEXITCODE
if (`$null -eq `$exitCode) { `$exitCode = 0 }
exit `$exitCode
"@

    Set-Content -LiteralPath $runnerPath -Value $runner -Encoding UTF8

    $output = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $runnerPath 2>&1
    $exitCode = $LASTEXITCODE
    $ended = Get-Date

    @(
        "name: $Name"
        "working_directory: $WorkingDirectory"
        "command:"
        $Command
        "exit_code: $exitCode"
        "started_at: $($started.ToString('o'))"
        "ended_at: $($ended.ToString('o'))"
        ""
        "output:"
        ($output | Out-String)
    ) | Set-Content -LiteralPath $logPath -Encoding UTF8

    $record = [ordered]@{
        name = $Name
        command = $Command
        working_directory = $WorkingDirectory
        exit_code = $exitCode
        started_at = $started.ToString('o')
        ended_at = $ended.ToString('o')
        duration_ms = [int](($ended - $started).TotalMilliseconds)
        log = "logs/$safeName.log"
    }
    ($record | ConvertTo-Json -Compress) | Add-Content -LiteralPath $Ctx.CommandsPath -Encoding UTF8
    $Ctx.CommandResults[$Name] = $record
    $script:LAST_CHECK_EXIT_CODE = $exitCode
}

function Add-FeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][ValidateSet('not_implemented','partial','full')][string]$Implementation,
        [Parameter(Mandatory=$true)][ValidateSet('not_tested','nonconformant','conformant')][string]$Conformance,
        [string[]]$Evidence = @(),
        [string]$Details = ''
    )

    $item = [ordered]@{
        id = $Id
        level = $Level
        category = $Category
        requirement = $Requirement
        implementation = $Implementation
        conformance = $Conformance
        evidence = @($Evidence)
        details = $Details
    }
    $Ctx.Assessments.Add($item) | Out-Null
}

function Add-CommandFeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][string]$CommandName,
        [string[]]$RequiredArtifacts = @(),
        [string]$Details = ''
    )

    $hasCommand = $Ctx.CommandResults.ContainsKey($CommandName)
    $exitCode = if ($hasCommand) { [int]$Ctx.CommandResults[$CommandName].exit_code } else { -999 }
    $missingArtifacts = @($RequiredArtifacts | Where-Object { -not (Test-Path -LiteralPath $_) })
    $implementation = 'not_implemented'
    $conformance = 'not_tested'

    if ($hasCommand) {
        $implementation = if ($exitCode -eq 0 -and $missingArtifacts.Count -eq 0) { 'full' } else { 'partial' }
        $conformance = if ($exitCode -eq 0 -and $missingArtifacts.Count -eq 0) { 'conformant' } else { 'nonconformant' }
    }

    $evidence = @()
    if ($hasCommand) {
        $evidence += [string]$Ctx.CommandResults[$CommandName].log
    }
    foreach ($artifact in $RequiredArtifacts) {
        if (Test-Path -LiteralPath $artifact) {
            $evidence += $artifact.Replace($Ctx.ResultDir, '').TrimStart('\')
        }
    }

    $detailParts = @()
    if ($Details) {
        $detailParts += $Details
    }
    if ($hasCommand) {
        $detailParts += "exit_code=$exitCode"
    } else {
        $detailParts += 'command was not executed'
    }
    if ($missingArtifacts.Count -gt 0) {
        $detailParts += "missing artifacts: $($missingArtifacts -join ', ')"
    }

    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance $conformance -Evidence $evidence -Details ($detailParts -join '; ')
}

function Add-BooleanFeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][bool]$Implemented,
        [Parameter(Mandatory=$true)][bool]$Conformant,
        [string[]]$Evidence = @(),
        [string]$Details = ''
    )

    $implementation = if ($Implemented) { 'full' } else { 'not_implemented' }
    $conformance = if (-not $Implemented) { 'not_tested' } elseif ($Conformant) { 'conformant' } else { 'nonconformant' }
    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance $conformance -Evidence $Evidence -Details $Details
}

function Add-SourceFeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][string[]]$Patterns,
        [ValidateSet('any','all')][string]$Match = 'all',
        [string]$Details = ''
    )

    $files = @(Get-ChildItem -LiteralPath $Ctx.RepoRoot -Recurse -File -ErrorAction SilentlyContinue | Where-Object {
        $_.FullName -notlike '*\.check-results\*' -and ($_.Extension -in @('.go', '.md') -or $_.Name -eq 'Makefile')
    })
    $matchedPatterns = @()
    $evidence = @()
    foreach ($pattern in $Patterns) {
        $hits = @($files | Select-String -Pattern $pattern -ErrorAction SilentlyContinue)
        if ($hits.Count -gt 0) {
            $matchedPatterns += $pattern
            $evidence += @($hits | Select-Object -First 5 | ForEach-Object {
                "$($_.Path):$($_.LineNumber)"
            })
        }
    }

    $implemented = if ($Match -eq 'all') {
        $matchedPatterns.Count -eq $Patterns.Count
    } else {
        $matchedPatterns.Count -gt 0
    }
    $implementation = if ($implemented) { 'partial' } else { 'not_implemented' }
    $detailText = "source-only check; matched=$($matchedPatterns.Count)/$($Patterns.Count)"
    if ($Details) {
        $detailText = "$Details; $detailText"
    }
    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance 'not_tested' -Evidence ($evidence | Select-Object -Unique) -Details $detailText
}

function Add-StandardEngineeringAssessments {
    param(
        [Parameter(Mandatory=$true)]$Ctx
    )

    $testFiles = @(Get-ChildItem -LiteralPath $Ctx.RepoRoot -Recurse -File -Filter '*_test.go' -ErrorAction SilentlyContinue | Where-Object { $_.FullName -notlike '*\.check-results\*' })
    $testFunctions = @($testFiles | Select-String -Pattern '^\s*func\s+Test[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)
    $benchmarkFunctions = @($testFiles | Select-String -Pattern '^\s*func\s+Benchmark[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)

    $testFileEvidence = @($testFiles | ForEach-Object { $_.FullName })
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.unit_tests_present' -Level 'engineering' -Category 'tests' -Requirement 'Go unit tests are present' -Implemented ($testFunctions.Count -gt 0) -Conformant ($testFunctions.Count -gt 0) -Evidence $testFileEvidence -Details "test_files=$($testFiles.Count); test_functions=$($testFunctions.Count)"
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.benchmarks_present' -Level 'engineering' -Category 'benchmarks' -Requirement 'Go benchmark tests are present' -Implemented ($benchmarkFunctions.Count -gt 0) -Conformant ($benchmarkFunctions.Count -gt 0) -Evidence $testFileEvidence -Details "benchmark_functions=$($benchmarkFunctions.Count)"

    if ($Ctx.CommandResults.ContainsKey('go_test_all')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.go_test_passes' -Level 'engineering' -Category 'tests' -Requirement 'go test ./... passes' -CommandName 'go_test_all'
    }
    if ($Ctx.CommandResults.ContainsKey('go_test_bench')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.benchmarks_run' -Level 'engineering' -Category 'benchmarks' -Requirement 'Benchmark tests run' -CommandName 'go_test_bench'
    }
    if ($Ctx.CommandResults.ContainsKey('go_test_race')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.race_test_passes' -Level 'engineering' -Category 'tests' -Requirement 'go test -race ./... passes' -CommandName 'go_test_race'
    }

    $readmePath = Join-Path $Ctx.RepoRoot 'README.md'
    $readmeOk = (Test-Path -LiteralPath $readmePath) -and ((Get-Item -LiteralPath $readmePath).Length -gt 100)
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.readme' -Level 'engineering' -Category 'documentation' -Requirement 'README.md exists and is not empty' -Implemented $readmeOk -Conformant $readmeOk -Evidence @('repo_snapshot/README.md')

    $makefilePath = Join-Path $Ctx.RepoRoot 'Makefile'
    $makefileText = if (Test-Path -LiteralPath $makefilePath) { Get-Content -LiteralPath $makefilePath -Raw } else { '' }
    foreach ($target in @('test','bench','demo')) {
        $targetOk = $makefileText -match "(?m)^\s*${target}\s*:"
        Add-BooleanFeatureAssessment -Ctx $Ctx -Id "engineering.make_$target" -Level 'engineering' -Category 'reproducibility' -Requirement "Makefile has target $target" -Implemented $targetOk -Conformant $targetOk -Evidence @('repo_snapshot/Makefile')
    }

    $controlPath = Join-Path $Ctx.RepoRoot 'testdata\control'
    $controlFiles = @()
    if (Test-Path -LiteralPath $controlPath) {
        $controlFiles = @(Get-ChildItem -LiteralPath $controlPath -Recurse -File -ErrorAction SilentlyContinue)
    }
    $controlEvidence = @($controlFiles | ForEach-Object { $_.FullName })
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.control_data' -Level 'engineering' -Category 'reproducibility' -Requirement 'Fixed testdata/control set exists' -Implemented ($controlFiles.Count -gt 0) -Conformant ($controlFiles.Count -gt 0) -Evidence $controlEvidence -Details "files=$($controlFiles.Count)"

    $solutionPath = Join-Path $Ctx.RepoRoot 'docs\reshenie.md'
    $solutionOk = (Test-Path -LiteralPath $solutionPath) -and ((Get-Item -LiteralPath $solutionPath).Length -gt 100)
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.solution_doc' -Level 'engineering' -Category 'documentation' -Requirement 'Non-empty docs/reshenie.md exists' -Implemented $solutionOk -Conformant $solutionOk -Evidence @('repo_snapshot/docs/reshenie.md')
}

function Copy-CheckPath {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Source,
        [Parameter(Mandatory=$true)][string]$RelativeDestination
    )

    if (-not (Test-Path -LiteralPath $Source)) {
        return
    }

    $destination = Join-Path $Ctx.ResultDir $RelativeDestination
    $parent = Split-Path -Parent $destination
    if ($parent) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
    Copy-Item -LiteralPath $Source -Destination $destination -Recurse -Force
}

function Complete-Check {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [hashtable]$Extra = @{}
    )

    Add-StandardEngineeringAssessments -Ctx $Ctx

    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_git_head' -Command "git rev-parse HEAD | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_head.txt' -Encoding UTF8" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_git_status' -Command "git status --short | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_status_short.txt' -Encoding UTF8" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_go_version' -Command "& '$($Ctx.GoCmd)' version | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_version.txt' -Encoding UTF8" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_go_env' -Command "& '$($Ctx.GoCmd)' env GOVERSION GOOS GOARCH | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_env.txt' -Encoding UTF8" | Out-Null

    foreach ($name in @('README.md', 'Makefile', 'go.mod', 'docs')) {
        $path = Join-Path $Ctx.RepoRoot $name
        Copy-CheckPath -Ctx $Ctx -Source $path -RelativeDestination "repo_snapshot/$name"
    }

    $assessmentItems = @($Ctx.Assessments)
    $assessmentSummary = [ordered]@{}
    foreach ($level in @('minimum','good','excellent','engineering')) {
        $items = @($assessmentItems | Where-Object { $_.level -eq $level })
        $assessmentSummary[$level] = [ordered]@{
            total = $items.Count
            full = @($items | Where-Object { $_.implementation -eq 'full' }).Count
            partial = @($items | Where-Object { $_.implementation -eq 'partial' }).Count
            not_implemented = @($items | Where-Object { $_.implementation -eq 'not_implemented' }).Count
            conformant = @($items | Where-Object { $_.conformance -eq 'conformant' }).Count
            nonconformant = @($items | Where-Object { $_.conformance -eq 'nonconformant' }).Count
            not_tested = @($items | Where-Object { $_.conformance -eq 'not_tested' }).Count
        }
    }
    Save-CheckJson -Path (Join-Path $Ctx.ResultDir 'assessment.json') -Value ([ordered]@{
        schema_version = 1
        statuses = [ordered]@{
            implementation = @('not_implemented','partial','full')
            conformance = @('not_tested','nonconformant','conformant')
        }
        summary = $assessmentSummary
        features = $assessmentItems
    })

    $manifest = [ordered]@{
        student = $Ctx.Student
        repo_root = $Ctx.RepoRoot
        started_at = $Ctx.StartedAt
        completed_at = (Get-Date).ToString('o')
        machine = [ordered]@{
            computer_name = $env:COMPUTERNAME
            user_name = $env:USERNAME
            os = (Get-CimInstance Win32_OperatingSystem).Caption
            powershell = $PSVersionTable.PSVersion.ToString()
        }
        result_dir = $Ctx.ResultDir
        commands_file = 'commands.jsonl'
        assessment_file = 'assessment.json'
        notes = $Extra
    }
    Save-CheckJson -Path (Join-Path $Ctx.ResultDir 'manifest.json') -Value $manifest

    $zipPath = "$($Ctx.ResultDir).zip"
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    Compress-Archive -Path (Join-Path $Ctx.ResultDir '*') -DestinationPath $zipPath -Force

    Write-Host "CHECK_RESULT_DIR=$($Ctx.ResultDir)"
    Write-Host "CHECK_RESULT_ZIP=$zipPath"
    return $zipPath
}


$ctx = New-CheckContext -Student 'logsan_check' -RepoRoot $RepoRoot -OutRoot $OutRoot

$logPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/app.log' -Content @'
2026-06-16 10:15:22 INFO user=ivanov email=ivanov@example.com ip=10.1.2.3 url=https://example.com/login token=ab12cd34ef56ab12cd34ef56ab12cd34
2026-06-16 10:16:01 WARN user=petrov email=petrov@example.com ip=10.1.2.3 path=C:\Users\Petrov\Documents\base.xlsx
2026-06-16 10:17:44 INFO normal log line without secrets
2026-06-16 10:18:10 INFO repeated email=ivanov@example.com open http://intranet.local/page
'@

$detectorsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/detectors.yaml' -Content @'
detectors:
  - detector_id: url
    type: regex
    pattern: 'https?://[^\s]+'
    replacement_prefix: url
    enabled: true
  - detector_id: email
    type: regex
    pattern: '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
    replacement_prefix: email
    enabled: true
  - detector_id: ipv4
    type: regex
    pattern: '\b(?:\d{1,3}\.){3}\d{1,3}\b'
    replacement_prefix: ip
    enabled: true
  - detector_id: token
    type: regex
    pattern: '\b[a-fA-F0-9]{24,}\b'
    replacement_prefix: token
    enabled: true
'@

Invoke-CheckCommand -Ctx $ctx -Name 'go_test_all' -Command "& '$($ctx.GoCmd)' test ./..."
Invoke-CheckCommand -Ctx $ctx -Name 'go_test_race' -Command "& '$($ctx.GoCmd)' test -race ./..."
Invoke-CheckCommand -Ctx $ctx -Name 'go_test_bench' -Command "& '$($ctx.GoCmd)' test -bench=. ./..."

if (Test-Path -LiteralPath (Join-Path $ctx.RepoRoot 'Makefile')) {
    Invoke-CheckCommand -Ctx $ctx -Name 'make_test' -Command 'make test'
    Invoke-CheckCommand -Ctx $ctx -Name 'make_bench' -Command 'make bench'
    Invoke-CheckCommand -Ctx $ctx -Name 'make_demo' -Command 'make demo'
}

$tool = Join-Path $ctx.OutputsDir 'logsan.exe'
Invoke-CheckCommand -Ctx $ctx -Name 'build_cli_cmd' -Command "& '$($ctx.GoCmd)' build -o '$tool' ./cmd/logsan"

$cleanLog = Join-Path $ctx.OutputsDir 'app.clean.log'
$jsonReport = Join-Path $ctx.OutputsDir 'report.json'
$dryReport = Join-Path $ctx.OutputsDir 'dry_run.md'
$mapping = Join-Path $ctx.OutputsDir 'mapping.json'

Invoke-CheckCommand -Ctx $ctx -Name 'cli_sanitize_file' -Command "& '$tool' sanitize --in '$logPath' --out '$cleanLog' --report '$jsonReport' --config '$detectorsPath' --outmap '$mapping'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_dry_run' -Command "& '$tool' dry-run --in '$logPath' --config '$detectorsPath' --report '$dryReport'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_sanitize_with_loaded_mapping' -Command "& '$tool' sanitize --in '$logPath' --out '$($ctx.OutputsDir)\app.clean.with-map.log' --report '$($ctx.OutputsDir)\report.with-map.json' --config '$detectorsPath' --inmap '$mapping'"

$expectedArtifacts = [ordered]@{
    clean_log = Test-Path -LiteralPath $cleanLog
    json_report = Test-Path -LiteralPath $jsonReport
    dry_run_report = Test-Path -LiteralPath $dryReport
    mapping = Test-Path -LiteralPath $mapping
}
Save-CheckJson -Path (Join-Path $ctx.OutputsDir 'artifact_presence.json') -Value $expectedArtifacts

Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.single_file' -Level 'minimum' -Category 'cli' -Requirement 'Single file processing' -CommandName 'cli_sanitize_file' -RequiredArtifacts @($cleanLog)
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.detectors' -Level 'minimum' -Category 'algorithm' -Requirement 'email ipv4 url detectors are applied' -CommandName 'cli_sanitize_file' -RequiredArtifacts @($cleanLog)
Add-SourceFeatureAssessment -Ctx $ctx -Id 'minimum.stable_replacement' -Level 'minimum' -Category 'algorithm' -Requirement 'Stable pseudonym replacement' -Patterns @('mapping|Mapping','replacement') -Match 'all'
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.json_report' -Level 'minimum' -Category 'format' -Requirement 'JSON replacement report' -CommandName 'cli_sanitize_file' -RequiredArtifacts @($jsonReport)
Add-SourceFeatureAssessment -Ctx $ctx -Id 'minimum.detector_tests' -Level 'minimum' -Category 'tests' -Requirement 'Detector unit tests exist' -Patterns @('Test.*Email','Test.*IP|Test.*IPv4','Test.*URL') -Match 'all'

Add-SourceFeatureAssessment -Ctx $ctx -Id 'good.directory_processing' -Level 'good' -Category 'cli' -Requirement 'Directory processing preserves structure' -Patterns @('ReadDir|WalkDir|Walk','MkdirAll') -Match 'all'
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.yaml_config' -Level 'good' -Category 'configuration' -Requirement 'YAML detector config' -CommandName 'cli_sanitize_file' -RequiredArtifacts @($cleanLog)
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.dry_run' -Level 'good' -Category 'cli' -Requirement 'dry-run command' -CommandName 'cli_dry_run' -RequiredArtifacts @($dryReport)
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.markdown_report' -Level 'good' -Category 'format' -Requirement 'dry-run Markdown report' -CommandName 'cli_dry_run' -RequiredArtifacts @($dryReport)
Add-SourceFeatureAssessment -Ctx $ctx -Id 'good.large_line_tests' -Level 'good' -Category 'tests' -Requirement 'Large line test exists' -Patterns @('Test.*Large|Test.*Long','strings\.Repeat') -Match 'all'

Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.mapping_save_load' -Level 'excellent' -Category 'algorithm' -Requirement 'Replacement mapping save and load' -CommandName 'cli_sanitize_with_loaded_mapping' -RequiredArtifacts @($mapping, (Join-Path $ctx.OutputsDir 'app.clean.with-map.log'))
Add-SourceFeatureAssessment -Ctx $ctx -Id 'excellent.parallel_files' -Level 'excellent' -Category 'performance' -Requirement 'Parallel file processing without races' -Patterns @('sync\.WaitGroup|errgroup','sync\.Mutex|sync\.RWMutex') -Match 'all'
Add-SourceFeatureAssessment -Ctx $ctx -Id 'excellent.safe_examples' -Level 'excellent' -Category 'report' -Requirement 'Safe replacement examples without originals' -Patterns @('replacement_examples|ReplacementExamples') -Match 'any'
Add-CommandFeatureAssessment -Ctx $ctx -Id 'excellent.large_benchmark' -Level 'excellent' -Category 'performance' -Requirement 'Key processing benchmark runs' -CommandName 'go_test_bench'

Complete-Check -Ctx $ctx -Extra @{
    expected_cli = 'logsan sanitize/dry-run'
    expected_outputs = @('app.clean.log', 'report.json', 'dry_run.md', 'mapping.json')
}


