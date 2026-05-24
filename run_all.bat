@echo off
chcp 65001 >nul
echo ========================================
echo Starting Receipt Processor Services
echo ========================================
echo.

echo [1/2] Starting Python Service...
cd /d %~dp0
start "Python Service" cmd /k "cd /d %~dp0python_service && python app.py"
timeout /t 3 /nobreak >nul

echo [2/2] Starting Go Backend...
cd /d %~dp0
start "Go Backend" cmd /k "cd /d %~dp0backend && go mod tidy && go run main.go"
timeout /t 2 /nobreak >nul

echo.
echo ========================================
echo Services started in separate windows
echo ========================================
echo Python Service: http://localhost:5000
echo Go Backend: http://localhost:8080
echo.
echo Waiting 8 seconds for services to initialize...
timeout /t 8 /nobreak >nul
echo.
echo Opening browser...
start http://localhost:8080
echo.
echo ========================================
echo Login: admin / admin123
echo ========================================
pause

