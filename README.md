# Система обработки квитанций

Веб-приложение для обработки и управления квитанциями об оплате с использованием OCR (оптическое распознавание символов).

## 📋 Описание

Система позволяет:
- Загружать квитанции (изображения, PDF)
- Автоматически извлекать данные из квитанций с помощью OCR
- Управлять группами студентов и банками
- Генерировать отчеты и экспортировать данные в Excel
- Управлять учебными годами и платежами

## 🏗️ Архитектура

Проект состоит из трех основных компонентов:

1. **Backend (Go)** - REST API сервер на порту `8080`
   - Управление пользователями и авторизация
   - CRUD операции для квитанций, групп, банков
   - Генерация отчетов
   - SQLite база данных

2. **Python Service** - OCR сервис на порту `5000`
   - Обработка изображений и PDF через EasyOCR
   - Извлечение данных из квитанций
   - Генерация Excel файлов

3. **Frontend** - Веб-интерфейс
   - Статические HTML/CSS/JavaScript файлы
   - Обслуживается через Go backend

## 📦 Требования

### Для Backend (Go)
- Go 1.21 или выше
- [Скачать Go](https://golang.org/dl/)

### Для Python Service
- Python 3.8 или выше
- pip (менеджер пакетов Python)

### Для Windows
- PowerShell или Command Prompt

## 🚀 Быстрый старт

### Вариант 1: Автоматический запуск (Windows)

1. Клонируйте репозиторий:
```bash
git clone <repository-url>
cd Kudaibergen
```

2. Запустите все сервисы одной командой:
```bash
run_all.bat
```

Скрипт автоматически:
- Запустит Python сервис на порту 5000
- Запустит Go backend на порту 8080
- Откроет браузер с приложением

**Учетные данные по умолчанию:**
- Логин: `admin`
- Пароль: `admin123`

### Вариант 2: Ручной запуск

#### Шаг 1: Установка зависимостей Python

```bash
cd python_service
pip install -r requirements.txt
```

**Примечание:** Установка EasyOCR может занять некоторое время, так как загружаются модели для распознавания.

#### Шаг 2: Запуск Python сервиса

```bash
cd python_service
python app.py
```

Сервис будет доступен на `http://localhost:5000`

#### Шаг 3: Установка зависимостей Go

```bash
cd backend
go mod download
```

#### Шаг 4: Запуск Go Backend

```bash
cd backend
go run main.go
```

Backend будет доступен на `http://localhost:8080`

#### Шаг 5: Открыть приложение

Откройте браузер и перейдите по адресу: `http://localhost:8080`

## 🔧 Настройка

### Изменение портов

**Backend (Go):**
Установите переменную окружения `PORT`:
```bash
# Windows
set PORT=3000
go run main.go

# Linux/Mac
export PORT=3000
go run main.go
```

**Python Service:**
Отредактируйте файл `python_service/app.py`, изменив порт в строке:
```python
app.run(host='0.0.0.0', port=5000, debug=True, use_reloader=False)
```

### База данных

База данных SQLite создается автоматически при первом запуске в файле `backend/receipts.db`.

**Важно:** Файл базы данных исключен из Git через `.gitignore`. При клонировании репозитория база будет создана заново с дефолтным администратором.

### Учетные данные администратора

По умолчанию создается администратор:
- **Логин:** `admin`
- **Пароль:** `admin123`

**Рекомендуется изменить пароль после первого входа!**

## 📁 Структура проекта

```
Kudaibergen/
├── backend/                 # Go backend
│   ├── handlers/           # HTTP обработчики
│   ├── models/             # Модели данных
│   ├── middleware/         # Middleware (авторизация)
│   ├── utils/              # Утилиты (JWT, пароли)
│   ├── main.go             # Точка входа
│   └── receipts.db         # База данных (создается автоматически)
│
├── python_service/         # Python OCR сервис
│   ├── app.py             # Flask приложение
│   ├── receipt_processor.py  # Обработка квитанций
│   ├── excel_generator.py    # Генерация Excel
│   ├── requirements.txt      # Python зависимости
│   ├── uploads/              # Временные файлы (исключены из Git)
│   └── exports/              # Экспортированные файлы (исключены из Git)
│
├── frontend/              # Веб-интерфейс
│   ├── index.html         # Главная страница
│   ├── app.js             # JavaScript логика
│   └── styles.css         # Стили
│
├── run_all.bat            # Скрипт запуска для Windows
├── go.mod                 # Go зависимости
├── go.sum                 # Go зависимости (checksums)
└── README.md              # Этот файл
```

## 🔌 API Endpoints

### Backend API (http://localhost:8080/api)

#### Публичные маршруты
- `POST /api/login` - Авторизация
- `GET /api/groups` - Список групп
- `GET /api/banks` - Список банков
- `GET /api/academic-years` - Список учебных годов
- `GET /api/academic-years/active` - Активный учебный год

#### Защищенные маршруты (требуют авторизации)
- `GET /api/receipts` - Список квитанций
- `POST /api/receipts` - Загрузка квитанции
- `GET /api/receipts/{id}` - Получить квитанцию
- `DELETE /api/receipts/{id}` - Удалить квитанцию
- `GET /api/receipts/{id}/file` - Получить файл квитанции

#### Административные маршруты (требуют роль admin)
- `GET /api/admin/users` - Список пользователей
- `POST /api/admin/users` - Создать пользователя
- `DELETE /api/admin/users/{id}` - Удалить пользователя
- `GET /api/admin/export` - Экспорт квитанций в Excel
- `POST /api/admin/groups` - Создать группу
- `PUT /api/admin/groups/{id}` - Обновить группу
- `DELETE /api/admin/groups/{id}` - Удалить группу
- `POST /api/admin/banks` - Создать банк
- `PUT /api/admin/banks/{id}` - Обновить банк
- `DELETE /api/admin/banks/{id}` - Удалить банк
- `POST /api/admin/academic-years` - Создать учебный год
- `PUT /api/admin/academic-years/{id}` - Обновить учебный год
- `DELETE /api/admin/academic-years/{id}` - Удалить учебный год
- `GET /api/admin/reports/group` - Отчет по группе
- `GET /api/admin/reports/group/export` - Экспорт отчета по группе
- `POST /api/admin/reports/payment` - Применить платеж
- `POST /api/admin/import/students` - Импорт студентов
- `GET /api/admin/import/template` - Шаблон для импорта

### Python Service API (http://localhost:5000)

- `POST /process` - Обработка квитанции через OCR
- `POST /export` - Генерация Excel файла
- `POST /export-report` - Генерация Excel отчета по группе
- `GET /health` - Проверка работоспособности

## 🛠️ Разработка

### Добавление новых зависимостей

**Go:**
```bash
cd backend
go get <package-name>
go mod tidy
```

**Python:**
```bash
cd python_service
pip install <package-name>
pip freeze > requirements.txt
```

### Сборка для production

**Go Backend:**
```bash
cd backend
go build -o receipt-processor main.go
./receipt-processor
```

**Python Service:**
Рекомендуется использовать production WSGI сервер, например Gunicorn:
```bash
pip install gunicorn
gunicorn -w 4 -b 0.0.0.0:5000 app:app
```

## 🐛 Решение проблем

### Python Service не запускается

1. Убедитесь, что Python установлен:
```bash
python --version
```

2. Установите зависимости:
```bash
cd python_service
pip install -r requirements.txt
```

3. Если возникают проблемы с EasyOCR, попробуйте установить отдельно:
```bash
pip install easyocr
```

### Go Backend не запускается

1. Убедитесь, что Go установлен:
```bash
go version
```

2. Обновите зависимости:
```bash
cd backend
go mod tidy
go mod download
```

3. Проверьте, что порт 8080 свободен

### База данных не создается

Убедитесь, что у приложения есть права на запись в директорию `backend/`.

### OCR не работает

1. Проверьте, что Python сервис запущен на порту 5000
2. Убедитесь, что модели EasyOCR загружены (первый запуск может занять время)
3. Проверьте формат загружаемого файла (поддерживаются изображения и PDF)

## 📝 Лицензия

[Укажите лицензию, если необходимо]

## 👥 Авторы

[Укажите авторов проекта]

## 🤝 Вклад в проект

1. Fork проекта
2. Создайте ветку для новой функции (`git checkout -b feature/AmazingFeature`)
3. Закоммитьте изменения (`git commit -m 'Add some AmazingFeature'`)
4. Запушьте в ветку (`git push origin feature/AmazingFeature`)
5. Откройте Pull Request

## 📞 Поддержка

Если у вас возникли вопросы или проблемы, создайте Issue в репозитории.
