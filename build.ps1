# �������� ������� Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go �� ����������. �������� � ���������� Go � https://golang.org/dl/"
    exit 1
}

# �������� ������� main.go
if (-not (Test-Path "main.go")) {
    Write-Error "main.go �� ������ � ������� ���������� (D:\Users\Vlad5\dev\keenetic-routes-via-vpn)"
    exit 1
}

# �������� ������� go.mod
if (-not (Test-Path "go.mod")) {
    Write-Host "go.mod �� ������. �������������� Go-������..."
    go mod init keenetic-routes-via-vpn
    if ($LASTEXITCODE -ne 0) {
        Write-Error "�� ������� ���������������� go.mod"
        exit 1
    }
    Write-Host "��������� ����������� gopkg.in/yaml.v3..."
    go get gopkg.in/yaml.v3
    if ($LASTEXITCODE -ne 0) {
        Write-Error "�� ������� �������� ����������� gopkg.in/yaml.v3"
        exit 1
    }
    go mod tidy
    if ($LASTEXITCODE -ne 0) {
        Write-Error "�� ������� ��������� go mod tidy"
        exit 1
    }
}

# �������� ���������� ��� ����������
if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
    if ($?) {
        Write-Host "������� ���������� bin"
    } else {
        Write-Error "�� ������� ������� ���������� bin"
        exit 1
    }
}

# ������ ��� MIPS
Write-Host "������ ��� MIPS..."
$env:GOOS="linux"; $env:GOARCH="mips"; $env:GOMIPS="softfloat"
go build -o bin/vpn-router-mips main.go
if ($LASTEXITCODE -ne 0) {
    Write-Error "�� ������� �������������� ��� MIPS"
    exit 1
}
Write-Host "��������� SHA256 ��� vpn-router-mips..."
Get-FileHash -Path bin/vpn-router-mips -Algorithm SHA256 | Select-Object -ExpandProperty Hash > bin/vpn-router-mips.sha256
if (-not $?) {
    Write-Error "�� ������� ������������� SHA256 ��� vpn-router-mips"
    exit 1
}

# ������ ��� MIPSel
Write-Host "������ ��� MIPSel..."
$env:GOOS="linux"; $env:GOARCH="mipsle"; $env:GOMIPS="softfloat"
go build -o bin/vpn-router-mipsel main.go
if ($LASTEXITCODE -ne 0) {
    Write-Error "�� ������� �������������� ��� MIPSel"
    exit 1
}
Write-Host "��������� SHA256 ��� vpn-router-mipsel..."
Get-FileHash -Path bin/vpn-router-mipsel -Algorithm SHA256 | Select-Object -ExpandProperty Hash > bin/vpn-router-mipsel.sha256
if (-not $?) {
    Write-Error "�� ������� ������������� SHA256 ��� vpn-router-mipsel"
    exit 1
}

# ������ ��� AArch64
Write-Host "������ ��� AArch64..."
$env:GOOS="linux"; $env:GOARCH="arm64"
go build -o bin/vpn-router-aarch64 main.go
if ($LASTEXITCODE -ne 0) {
    Write-Error "�� ������� �������������� ��� AArch64"
    exit 1
}
Write-Host "��������� SHA256 ��� vpn-router-aarch64..."
Get-FileHash -Path bin/vpn-router-aarch64 -Algorithm SHA256 | Select-Object -ExpandProperty Hash > bin/vpn-router-aarch64.sha256
if (-not $?) {
    Write-Error "�� ������� ������������� SHA256 ��� vpn-router-aarch64"
    exit 1
}

Write-Host "������ ��������� �������!"
Write-Host "��������� � �� SHA256 checksum-����� ��������� � D:\Users\Vlad5\dev\keenetic-routes-via-vpn\bin:"
Get-ChildItem -Path bin

Write-Host "`n��������� ����:"
Write-Host "1. ���������� ����������� ������ ������� Keenetic (mips, mipsel ��� aarch64) � ������� 'uname -m' �� �������:"
Write-Host "   ssh root@<router-ip> 'uname -m'"
Write-Host "2. ���������� ��������������� �������� � ��� checksum �� ������:"
Write-Host "   scp bin/vpn-router-<arch> bin/vpn-router-<arch>.sha256 root@<router-ip>:/opt/bin/"
Write-Host "3. ��������� ����������� ��������� �� �������:"
Write-Host "   ssh root@<router-ip> 'sha256sum /opt/bin/vpn-router | cut -d\" \" -f1 > /opt/bin/computed.sha256; cmp /opt/bin/computed.sha256 /opt/bin/vpn-router-<arch>.sha256'"
Write-Host "4. ���������, ��� �������� ����� ����� �� ����������:"
Write-Host "   ssh root@<router-ip> 'chmod +x /opt/bin/vpn-router'"
Write-Host "5. ���������� ��������� � ������� install.sh (��� ����������)."
Write-Host "����������: SHA256-����� �������� ���������, ��� �������� �� ���������. ��������� �� � GitHub Releases ������ � �����������."