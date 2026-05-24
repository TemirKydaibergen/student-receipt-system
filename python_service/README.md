# Python Service для обработки квитанций

## Установка

```bash
pip install -r requirements.txt
```

## Запуск

```bash
python app.py
```

Сервис будет доступен на `http://localhost:5000`

## API Endpoints

- `POST /process` - обработка квитанции через OCR
- `POST /export` - генерация Excel файла
- `GET /health` - проверка работоспособности

