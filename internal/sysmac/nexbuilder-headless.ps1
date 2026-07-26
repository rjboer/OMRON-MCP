param(
    [Parameter(Mandatory=$true)][string]$ProjectDirectory,
    [Parameter(Mandatory=$true)][string]$OutputDirectory,
    [ValidateSet('NJ5','NX1','NX5','NX7','NY5')][string]$Target = 'NX5'
)

$ErrorActionPreference = 'Stop'
$install = 'C:\Program Files\OMRON\Sysmac Studio'
$builderDir = Join-Path $install 'builder2'
$interfaceDll = Join-Path $install 'Modules\Nex\INexBuilder2.dll'
$builderDll = Join-Path $builderDir 'NexBuilder2.dll'
[Reflection.Assembly]::LoadFrom($interfaceDll) | Out-Null
$assembly = [Reflection.Assembly]::LoadFrom($builderDll)
$builderType = $assembly.GetType('Omron.NexBuilder2.NexBuilder')
$platformType = ([Reflection.Assembly]::LoadFrom($interfaceDll)).GetType('Omron.NexBuilder2.Platform')
$platform = [Enum]::Parse($platformType, 'Platform_All')

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$cxils = Get-ChildItem -LiteralPath $ProjectDirectory -Filter '*.cxil2' -File | Sort-Object Name
if ($cxils.Count -eq 0) { throw "No .cxil2 files found in $ProjectDirectory" }

$builder = [Activator]::CreateInstance($builderType, @($Target, 1))
$records = @()
try {
    foreach ($cxil in $cxils) { $builder.AddCxIL($cxil.FullName) }
    $results = $builder.Start($ProjectDirectory, $OutputDirectory, $platform)
    $builder.Join()
    foreach ($result in $results.GetResults()) {
        $messages = @()
        foreach ($message in $result.GetMessages()) {
            $messages += [pscustomobject]@{
                code = $message.GetCode()
                y = $message.GetY()
                message = $message.GetMessage()
                file = $message.GetFilePath()
                hints = @($message.GetHint())
            }
        }
        $records += [pscustomobject]@{
            file = $result.GetFileName()
            status = $result.GetStatus().ToString()
            messages = $messages
        }
    }
}
finally { $builder.Dispose() }

[pscustomobject]@{
    project_directory = $ProjectDirectory
    output_directory = $OutputDirectory
    target = $Target
    inputs = @($cxils.FullName)
    results = $records
    generated_cxif2 = @(Get-ChildItem -LiteralPath $OutputDirectory -Filter '*.cxif2' -File -ErrorAction SilentlyContinue | % FullName)
} | ConvertTo-Json -Depth 8
