
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

# Сборка для MIPSel
Write-Host "Сборка для MIPSel..."
$env:GOOS="linux"; $env:GOARCH="mipsle"; $env:GOMIPS="softfloat"
go build -o bin/vpn-router-mipsel main.go
if ($LASTEXITCODE -ne 0) {
    Write-Error "Не удалось скомпилировать для MIPSel"
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

Write-Host "Сборка завершена успешно!"
Write-Host "Бинарники находятся в D:\Users\Vlad5\dev\keenetic-routes-via-vpn\bin:"
Get-ChildItem -Path bin

Write-Host "`nСледующие шаги:"
Write-Host "1. Определите архитектуру вашего роутера Keenetic (mips, mipsel или aarch64) с помощью 'uname -m' на роутере:"
Write-Host "   ssh root@<router-ip> 'uname -m'"
Write-Host "2. Скопируйте соответствующий бинарник на роутер:"
Write-Host "   scp bin/vpn-router-<arch> root@<router-ip>:/opt/bin/vpn-router"
Write-Host "3. Убедитесь, что бинарник имеет права на выполнение:"
Write-Host "   ssh root@<router-ip> 'chmod +x /opt/bin/vpn-router'"
Write-Host "4. Продолжите установку с помощью install.sh (без компиляции)."
