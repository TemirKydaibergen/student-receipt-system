import os
import sys

# Отключаем автоматическую загрузку .env файла ДО импорта Flask
os.environ['FLASK_SKIP_DOTENV'] = '1'
os.environ['FLASK_APP'] = 'app.py'

from flask import Flask, request, jsonify, send_file
from flask_cors import CORS
import json
import re
from datetime import datetime
from receipt_processor import ReceiptProcessor
from excel_generator import ExcelGenerator

app = Flask(__name__)
CORS(app)

UPLOAD_FOLDER = 'uploads'
EXPORT_FOLDER = 'exports'

os.makedirs(UPLOAD_FOLDER, exist_ok=True)
os.makedirs(EXPORT_FOLDER, exist_ok=True)

processor = ReceiptProcessor()
excel_gen = ExcelGenerator()

@app.route('/process', methods=['POST'])
def process_receipt():
    """Обработка квитанции через OCR"""
    try:
        if 'file' not in request.files:
            return jsonify({'error': 'No file provided'}), 400

        file = request.files['file']
        bank_type = request.form.get('bank_type', 'Halyk')
        auto_extract_bank = request.form.get('auto_extract_bank', 'false') == 'true'
        auto_extract_fullname = request.form.get('auto_extract_fullname', 'false') == 'true'
        receipt_id = request.form.get('receipt_id')

        if file.filename == '':
            return jsonify({'error': 'No file selected'}), 400

        # Сохранение временного файла
        # Берем только имя файла, без пути
        original_filename = os.path.basename(file.filename)
        filename = f"temp_{receipt_id}_{original_filename}"
        filepath = os.path.join(UPLOAD_FOLDER, filename)
        
        # Убеждаемся, что директория существует
        os.makedirs(UPLOAD_FOLDER, exist_ok=True)
        
        file.save(filepath)
        print(f"File saved to: {filepath}, size: {os.path.getsize(filepath)} bytes")

        # Обработка квитанции
        # Если auto_extract_bank=True, передаем None для автоматического определения
        if auto_extract_bank:
            process_bank_type = None
        else:
            process_bank_type = bank_type if bank_type else 'Halyk'
        
        print(f"Processing receipt: filepath={filepath}, bank_type={process_bank_type}, auto_extract_fullname={auto_extract_fullname}")
        result = processor.process_receipt(filepath, process_bank_type, auto_extract_fullname)
        print(f"Processing result: {result}")

        # Удаление временного файла ПОСЛЕ обработки
        # НЕ удаляем файл до OCR, так как он может понадобиться для обработки
        try:
            if os.path.exists(filepath):
                os.remove(filepath)
                print(f"Temporary file removed: {filepath}")
        except Exception as e:
            print(f"Warning: Failed to remove temporary file {filepath}: {e}")
            pass  # Игнорируем ошибки удаления

        return jsonify(result), 200

    except Exception as e:
        import traceback
        error_msg = str(e)
        error_trace = traceback.format_exc()
        print(f"Error processing receipt: {error_msg}")
        print(f"Traceback: {error_trace}")
        return jsonify({'error': error_msg, 'traceback': error_trace}), 500

@app.route('/export', methods=['POST'])
def export_excel():
    """Генерация Excel файла"""
    try:
        receipts_data = request.json

        if not receipts_data:
            return jsonify({'error': 'No data provided'}), 400

        # Генерация Excel
        excel_path = excel_gen.generate_excel(receipts_data)

        return send_file(
            excel_path,
            mimetype='application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
            as_attachment=True,
            download_name='receipts.xlsx'
        )

    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/export-report', methods=['POST'])
def export_group_report():
    """Генерация Excel файла отчета по группе"""
    try:
        report_data = request.json

        if not report_data:
            return jsonify({'error': 'No data provided'}), 400

        # Генерация Excel отчета
        excel_path = excel_gen.generate_group_report(report_data)

        return send_file(
            excel_path,
            mimetype='application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
            as_attachment=True,
            download_name='group_report.xlsx'
        )

    except Exception as e:
        import traceback
        print(f"Error exporting report: {e}")
        print(f"Traceback: {traceback.format_exc()}")
        return jsonify({'error': str(e), 'traceback': traceback.format_exc()}), 500

@app.route('/health', methods=['GET'])
def health():
    """Проверка работоспособности сервиса"""
    return jsonify({'status': 'ok'}), 200

if __name__ == '__main__':
    # Используем werkzeug напрямую чтобы обойти проблему с dotenv
    from werkzeug.serving import run_simple
    try:
        # Пробуем обычный способ
        app.run(host='0.0.0.0', port=5000, debug=True, use_reloader=False)
    except (UnicodeDecodeError, Exception) as e:
        # Если ошибка, используем werkzeug напрямую
        print(f"Note: Using werkzeug directly (dotenv issue bypassed)")
        run_simple('0.0.0.0', 5000, app, use_reloader=False, use_debugger=True)

