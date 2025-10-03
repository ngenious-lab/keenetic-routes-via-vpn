# Проверка наличия Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go не установлен. Скачайте и установите Go с https://golang.org/dl/"
    exit 1
}

# Проверка наличия main.go
if (-not (Test-Path "main.go")) {
    Write-Error "main.go не найден в текущей директории (D:\Users\Vlad5\dev\keenetic-routes-via-vpn)"
    exit 1
}

# Проверка наличия go.mod
if (-not (Test-Path "go.mod")) {
    Write-Host "go.mod не найден. Инициализируем Go-модуль..."
    go mod init keenetic-routes-via-vpn
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Не удалось инициализировать go.mod"
        exit 1
    }
    Write-Host "Добавляем зависимость gopkg.in/yaml.v3..."
    go get gopkg.in/yaml.v3
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Не удалось добавить зависимость gopkg.in/yaml.v3"
        exit 1
    }
    go mod tidy
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Не удалось выполнить go mod tidy"
        exit 1
    }
}

# Создание директории для бинарников
if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
    if ($?) {
        Write-Host "Создана директория bin"
    } else {
        Write-Error "Не удалось создать директорию bin"
        exit 1
    }
}

# Сборка для MIPS
Write-Host "Сборка для MIPS..."
$env:GOOS="linux"; $env:GOARCH="mips"; $env:GOMIPS="softfloat"
go build -o bin/vpn-router-mips main.go
if ($LASTEXITCODE -ne 0) {
    Write-Error "Не удалось скомпилировать для MIPS"
    exit 1
}
Write-Host "Генерация SHA256 для vpn-router-mips..."
Get-FileHash -Path bin/vpn-router-mips -Algorithm SHA256 | Select-Object -ExpandProperty Hash > bin/vpn-router-mips.sha256
if (-not $?) {
    Write-Error "Не удалось сгенерировать SHA256 для vpn-router-mips"
    exit 1
}

# Сборка для MIPSel
Write-Host "Сборка для MIPSel..."
$env:GOOS="linux"; $env:GOARCH="mipsle"; $env:GOMIPS="softfloat"
go build -o bin/vpn-router-mipsel main.go
if ($LASTEXITCODE -ne 0) {
    Write-Error "Не удалось скомпилировать для MIPSel"
    exit 1
}
Write-Host "Генерация SHA256 для vpn-router-mipsel..."
Get-FileHash -Path bin/vpn-router-mipsel -Algorithm SHA256 | Select-Object -ExpandProperty Hash > bin/vpn-router-mipsel.sha256
if (-not $?) {
    Write-Error "Не удалось сгенерировать SHA256 для vpn-router-mipsel"
    exit 1
}

# Сборка для AArch64
Write-Host "Сборка для AArch64..."
$env:GOOS="linux"; $env:GOARCH="arm64"
go build -o bin/vpn-router-aarch64 main.go
if ($LASTEXITCODE -ne 0) {
    Write-Error "Не удалось скомпилировать для AArch64"
    exit 1
}
Write-Host "Генерация SHA256 для vpn-router-aarch64..."
Get-FileHash -Path bin/vpn-router-aarch64 -Algorithm SHA256 | Select-Object -ExpandProperty Hash > bin/vpn-router-aarch64.sha256
if (-not $?) {
    Write-Error "Не удалось сгенерировать SHA256 для vpn-router-aarch64"
    exit 1
}

Write-Host "Сборка завершена успешно!"
Write-Host "Бинарники и их SHA256 checksum-файлы находятся в D:\Users\Vlad5\dev\keenetic-routes-via-vpn\bin:"
Get-ChildItem -Path bin

Write-Host "`nСледующие шаги:"
Write-Host "1. Определите архитектуру вашего роутера Keenetic (mips, mipsel или aarch64) с помощью 'uname -m' на роутере:"
Write-Host "   ssh root@<router-ip> 'uname -m'"
Write-Host "2. Скопируйте соответствующий бинарник и его checksum на роутер:"
Write-Host "   scp bin/vpn-router-<arch> bin/vpn-router-<arch>.sha256 root@<router-ip>:/opt/bin/"
Write-Host "3. Проверьте целостность бинарника на роутере:"
Write-Host "   ssh root@<router-ip> 'sha256sum /opt/bin/vpn-router | cut -d\" \" -f1 > /opt/bin/computed.sha256; cmp /opt/bin/computed.sha256 /opt/bin/vpn-router-<arch>.sha256'"
Write-Host "4. Убедитесь, что бинарник имеет права на выполнение:"
Write-Host "   ssh root@<router-ip> 'chmod +x /opt/bin/vpn-router'"
Write-Host "5. Продолжите установку с помощью install.sh (без компиляции)."
Write-Host "Примечание: SHA256-файлы помогают убедиться, что бинарник не поврежден. Загрузите их в GitHub Releases вместе с бинарниками."