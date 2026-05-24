import easyocr
import re
from datetime import datetime
from PIL import Image, ImageEnhance, ImageFilter
import os
import numpy as np

# Попытка импорта библиотек для PDF
try:
    import pdf2image
    PDF2IMAGE_AVAILABLE = True
except ImportError:
    PDF2IMAGE_AVAILABLE = False

try:
    import fitz  # PyMuPDF
    PYMUPDF_AVAILABLE = True
except ImportError:
    PYMUPDF_AVAILABLE = False

class ReceiptProcessor:
    def __init__(self):
        # Инициализация EasyOCR (поддерживает русский и английский)
        # Это может занять время при первом запуске (загрузка моделей)
        try:
            print("Initializing EasyOCR...")
            self.reader = easyocr.Reader(['ru', 'en'], gpu=False)
            print("EasyOCR initialized successfully")
        except Exception as e:
            print(f"Error initializing EasyOCR: {e}")
            raise

    def process_receipt(self, filepath, bank_type=None, auto_extract_fullname=False):
        """
        Обработка квитанции и извлечение данных
        """
        original_pdf_path = None  # Сохраняем путь к оригинальному PDF для удаления
        converted_pdf_path = None  # Сохраняем путь к конвертированному PDF для удаления
        try:
            if not os.path.exists(filepath):
                raise FileNotFoundError(f"File not found: {filepath}")
            
            print(f"Processing receipt: {filepath}, bank_type: {bank_type}")
            # Конвертация PDF в изображение, если необходимо
            if filepath.lower().endswith('.pdf'):
                original_pdf_path = filepath  # Сохраняем путь к оригинальному PDF
                # Используем UUID для имени файла, чтобы избежать проблем с кодировкой
                import uuid
                base_dir = os.path.dirname(filepath) if os.path.dirname(filepath) else '.'
                unique_id = str(uuid.uuid4())[:8]
                image_path = os.path.join(base_dir, f"pdf_converted_{unique_id}.png")
                
                converted = False
                error_messages = []
                
                # Сначала пробуем PyMuPDF (быстрее и проще для Windows)
                if PYMUPDF_AVAILABLE:
                    try:
                        doc = fitz.open(filepath)
                        if len(doc) > 0:
                            page = doc[0]
                            pix = page.get_pixmap(matrix=fitz.Matrix(2, 2))  # Увеличиваем разрешение
                            pix.save(image_path)
                            converted_pdf_path = image_path  # Сохраняем путь для удаления
                            filepath = image_path
                            doc.close()
                            converted = True
                            print(f"PDF converted to image using PyMuPDF: {image_path}")
                    except Exception as e:
                        error_messages.append(f"PyMuPDF error: {str(e)}")
                        print(f"PyMuPDF conversion failed: {e}")
                
                # Если PyMuPDF не сработал, пробуем pdf2image
                if not converted and PDF2IMAGE_AVAILABLE:
                    try:
                        images = pdf2image.convert_from_path(filepath, dpi=200)
                        if images:
                            images[0].save(image_path, 'PNG')
                            converted_pdf_path = image_path  # Сохраняем путь для удаления
                            filepath = image_path
                            converted = True
                            print(f"PDF converted to image using pdf2image: {image_path}")
                    except Exception as e:
                        error_messages.append(f"pdf2image error: {str(e)}")
                        print(f"pdf2image conversion failed: {e}")
                
                if not converted:
                    error_msg = "Не удалось обработать PDF. "
                    if not PYMUPDF_AVAILABLE and not PDF2IMAGE_AVAILABLE:
                        error_msg += "Установите PyMuPDF (pip install pymupdf) или pdf2image (pip install pdf2image). "
                        error_msg += "Для pdf2image на Windows также требуется poppler (https://github.com/oschwartz10612/poppler-windows/releases)"
                    else:
                        error_msg += "Ошибки: " + "; ".join(error_messages)
                    raise Exception(error_msg)

            # Предобработка изображения для лучшего распознавания (для JPG/JPEG и других изображений)
            image_extensions = ['.jpg', '.jpeg', '.png', '.bmp', '.tiff', '.tif', '.gif']
            is_image = any(filepath.lower().endswith(ext) for ext in image_extensions)
            
            print(f"File type check: is_image={is_image}, filepath={filepath}")
            print(f"File exists before preprocessing: {os.path.exists(filepath)}")
            
            processed_filepath = filepath
            if is_image:
                try:
                    print(f"Preprocessing image: {filepath}")
                    # Проверяем, что файл существует
                    if not os.path.exists(filepath):
                        raise FileNotFoundError(f"Image file not found: {filepath}")
                    
                    # Открываем изображение
                    img = Image.open(filepath)
                    print(f"Original image mode: {img.mode}, size: {img.size}")
                    
                    # Конвертируем в RGB, если необходимо
                    if img.mode != 'RGB':
                        print(f"Converting image from {img.mode} to RGB")
                        img = img.convert('RGB')
                    
                    # Увеличиваем разрешение изображения для лучшего OCR (как для PDF)
                    # PDF конвертируется с разрешением 2x, делаем то же для изображений
                    original_size = img.size
                    # Увеличиваем до минимум 2000px по большей стороне, если меньше
                    max_dimension = max(original_size)
                    if max_dimension < 2000:
                        scale_factor = 2000 / max_dimension
                        new_size = (int(original_size[0] * scale_factor), int(original_size[1] * scale_factor))
                        img = img.resize(new_size, Image.Resampling.LANCZOS)
                        print(f"Image resized from {original_size} to {new_size} (scale: {scale_factor:.2f})")
                    
                    # Улучшаем контраст и резкость для лучшего распознавания
                    # Для JPG/JPEG и PNG увеличиваем контраст больше, так как они часто имеют низкий контраст
                    is_jpg = filepath.lower().endswith('.jpg') or filepath.lower().endswith('.jpeg')
                    is_png = filepath.lower().endswith('.png')
                    
                    # Более агрессивная обработка для изображений (как для PDF)
                    # Для PNG/JPG используем максимально агрессивную обработку
                    contrast_factor = 2.2 if (is_jpg or is_png) else 1.8  # Еще больше контраста для PNG/JPG
                    sharpness_factor = 1.8 if (is_jpg or is_png) else 1.5  # Еще больше резкости для PNG/JPG
                    
                    # Первое усиление контраста
                    enhancer = ImageEnhance.Contrast(img)
                    img = enhancer.enhance(contrast_factor)
                    print(f"Contrast enhanced by factor {contrast_factor}")
                    
                    # Усиление резкости
                    enhancer = ImageEnhance.Sharpness(img)
                    img = enhancer.enhance(sharpness_factor)
                    print(f"Sharpness enhanced by factor {sharpness_factor}")
                    
                    # Дополнительная обработка для JPG/PNG: увеличиваем яркость
                    if is_jpg or is_png:
                        enhancer = ImageEnhance.Brightness(img)
                        img = enhancer.enhance(1.2)  # Увеличиваем яркость больше
                        print("Brightness enhanced for image")
                        
                        # Дополнительное усиление контраста для PNG/JPG
                        enhancer = ImageEnhance.Contrast(img)
                        img = enhancer.enhance(1.3)  # Дополнительный контраст
                        print("Additional contrast enhancement for PNG/JPG")
                    
                    # Применяем фильтр для уменьшения шума
                    try:
                        # Легкое размытие для уменьшения шума, затем снова увеличиваем резкость
                        img = img.filter(ImageFilter.MedianFilter(size=3))
                        enhancer = ImageEnhance.Sharpness(img)
                        img = enhancer.enhance(1.3)  # Финальная резкость (увеличена)
                        print("Noise reduction and final sharpening applied")
                    except Exception as e:
                        print(f"Filter application warning: {e}")
                    
                    # Для PNG/JPG применяем дополнительное улучшение четкости
                    if is_jpg or is_png:
                        try:
                            # Применяем фильтр повышения резкости
                            img = img.filter(ImageFilter.SHARPEN)
                            print("Additional sharpening filter applied")
                        except Exception as e:
                            print(f"Sharpening filter warning: {e}")
                    
                    # Сохраняем обработанное изображение во временный файл с высоким качеством
                    # Используем только ASCII символы в имени файла, чтобы избежать проблем с кодировкой
                    import uuid
                    base_dir = os.path.dirname(filepath) if os.path.dirname(filepath) else '.'
                    # Создаем уникальное имя файла без кириллицы
                    unique_id = str(uuid.uuid4())[:8]
                    processed_filepath = os.path.join(base_dir, f"processed_{unique_id}.png")
                    
                    # Убеждаемся, что директория существует
                    os.makedirs(base_dir, exist_ok=True)
                    
                    # Сохраняем с максимальным качеством
                    img.save(processed_filepath, 'PNG', quality=100, optimize=False)
                    print(f"Image preprocessed and saved to: {processed_filepath}")
                    print(f"Processed file exists: {os.path.exists(processed_filepath)}")
                    print(f"Processed file size: {os.path.getsize(processed_filepath) if os.path.exists(processed_filepath) else 0} bytes")
                    print(f"Processed image size: {img.size}")
                except Exception as e:
                    import traceback
                    print(f"Image preprocessing failed, using original: {e}")
                    print(f"Preprocessing traceback: {traceback.format_exc()}")
                    processed_filepath = filepath
            
            # OCR распознавание
            # Используем абсолютный путь для избежания проблем с кодировкой
            processed_filepath_abs = os.path.abspath(processed_filepath)
            print(f"Running OCR on {processed_filepath_abs}...")
            print(f"File exists: {os.path.exists(processed_filepath_abs)}")
            if not os.path.exists(processed_filepath_abs):
                raise FileNotFoundError(f"Processed file not found: {processed_filepath_abs}")
            
            # Проверяем размер файла
            file_size = os.path.getsize(processed_filepath_abs)
            print(f"File size: {file_size} bytes")
            
            # Используем абсолютный путь для OCR
            ocr_filepath = processed_filepath_abs
            if file_size == 0:
                raise Exception("Processed image file is empty")
            
            # Загружаем изображение и конвертируем в numpy array для OCR
            # Это решает проблемы с кодировкой путей на Windows
            try:
                img = Image.open(ocr_filepath)
                # Конвертируем в RGB, если необходимо
                if img.mode != 'RGB':
                    img = img.convert('RGB')
                # Конвертируем PIL Image в numpy array для EasyOCR
                img_array = np.array(img)
                print(f"Image file loaded successfully: {ocr_filepath}, size: {img.size}, array shape: {img_array.shape}")
            except Exception as e:
                print(f"ERROR: Cannot open image file: {e}")
                import traceback
                print(f"Image loading traceback: {traceback.format_exc()}")
                raise Exception(f"Cannot read image file: {e}")
            
            try:
                # Улучшенные параметры OCR для лучшего распознавания
                # Для изображений (особенно JPG) используем более мягкие параметры
                is_image_file = any(processed_filepath.lower().endswith(ext) for ext in ['.jpg', '.jpeg', '.png', '.bmp', '.tiff', '.tif', '.gif'])
                
                if is_image_file:
                    # Для изображений используем максимально мягкие параметры для лучшего распознавания
                    # Пробуем несколько вариантов OCR с разными параметрами
                    print("Attempting OCR with relaxed parameters for image...")
                    results = []
                    
                    # Первая попытка: очень мягкие параметры
                    # Используем numpy array вместо пути к файлу для лучшей совместимости
                    try:
                        print(f"Attempting OCR on numpy array (shape: {img_array.shape})")
                        results = self.reader.readtext(
                            img_array,  # Передаем numpy array вместо пути
                            detail=1,  # Полная детализация
                            paragraph=False,  # Не группировать в параграфы
                            width_ths=0.3,  # Очень низкий порог для объединения
                            height_ths=0.3,  # Очень низкий порог для объединения
                            allowlist=None,  # Разрешаем все символы
                            blocklist=None,  # Не блокируем символы
                            text_threshold=0.3,  # Низкий порог уверенности для текста
                            link_threshold=0.3,  # Низкий порог для связывания символов
                        )
                        print(f"First OCR attempt found {len(results)} text regions")
                        if len(results) > 0:
                            print(f"First 3 results: {results[:3]}")
                    except Exception as e:
                        print(f"First OCR attempt failed: {e}")
                        import traceback
                        print(f"First OCR attempt traceback: {traceback.format_exc()}")
                    
                    # Если первая попытка не дала результатов, пробуем еще более мягкие параметры
                    if len(results) == 0:
                        print("Trying OCR with even more relaxed parameters...")
                        try:
                            results = self.reader.readtext(
                                img_array,  # Используем numpy array
                                detail=1,
                                paragraph=False,
                                width_ths=0.2,  # Минимальный порог
                                height_ths=0.2,  # Минимальный порог
                                allowlist=None,
                                blocklist=None,
                                text_threshold=0.2,  # Очень низкий порог
                                link_threshold=0.2,  # Очень низкий порог
                            )
                            print(f"Second OCR attempt found {len(results)} text regions")
                        except Exception as e:
                            print(f"Second OCR attempt failed: {e}")
                    
                    # Если все еще нет результатов, пробуем без порогов
                    if len(results) == 0:
                        print("Trying OCR without thresholds...")
                        try:
                            results = self.reader.readtext(
                                img_array,  # Используем numpy array
                                detail=1,
                                paragraph=False,
                            )
                            print(f"Third OCR attempt (no thresholds) found {len(results)} text regions")
                        except Exception as e:
                            print(f"Third OCR attempt failed: {e}")
                else:
                    # Для PDF используем стандартные параметры
                    # Загружаем изображение для PDF тоже
                    try:
                        pdf_img = Image.open(ocr_filepath)
                        if pdf_img.mode != 'RGB':
                            pdf_img = pdf_img.convert('RGB')
                        pdf_img_array = np.array(pdf_img)
                        print(f"PDF image loaded, array shape: {pdf_img_array.shape}")
                    except Exception as e:
                        print(f"WARNING: Cannot load PDF image as array, using file path: {e}")
                        pdf_img_array = ocr_filepath
                    
                    results = self.reader.readtext(
                        pdf_img_array,  # Используем numpy array или путь
                        detail=1,
                        paragraph=False,
                        width_ths=0.7,
                        height_ths=0.7,
                    )
                
                print(f"OCR completed, found {len(results)} text regions")
                
                # Логируем первые несколько распознанных фрагментов для отладки
                if len(results) > 0:
                    print("First 10 OCR results:")
                    for i, result in enumerate(results[:10]):
                        confidence = result[2] if len(result) > 2 else 0.0
                        print(f"  {i+1}. Text: '{result[1]}', Confidence: {confidence:.2f}")
                else:
                    print("WARNING: OCR returned no results!")
                    
            except Exception as e:
                print(f"OCR error: {e}")
                import traceback
                print(f"OCR traceback: {traceback.format_exc()}")
                # Удаляем временный файл даже при ошибке
                if processed_filepath != filepath and os.path.exists(processed_filepath):
                    try:
                        os.remove(processed_filepath)
                    except:
                        pass
                raise Exception(f"OCR failed: {str(e)}")

            # Извлечение текста с фильтрацией по уверенности
            # Для изображений используем очень низкий порог, так как они могут иметь низкое качество
            is_image_file = any(processed_filepath.lower().endswith(ext) for ext in ['.jpg', '.jpeg', '.png', '.bmp', '.tiff', '.tif', '.gif'])
            # Для изображений используем очень низкий порог уверенности
            confidence_threshold = 0.1 if is_image_file else 0.15  # Очень низкий порог для изображений
            
            filtered_results = [r for r in results if len(r) > 2 and r[2] > confidence_threshold]
            print(f"Filtered results: {len(filtered_results)} out of {len(results)} (threshold: {confidence_threshold})")
            
            # Логируем все результаты для отладки
            if len(results) > 0:
                print("All OCR results (before filtering):")
                for i, result in enumerate(results[:30]):  # Первые 30 результатов для лучшей отладки
                    confidence = result[2] if len(result) > 2 else 0.0
                    text = result[1] if len(result) > 1 else ''
                    print(f"  {i+1}. Text: '{text[:80]}', Confidence: {confidence:.2f}")
            
            full_text = ' '.join([result[1] for result in filtered_results])
            print(f"Extracted text length: {len(full_text)} characters")
            if len(full_text) > 0:
                print(f"Full extracted text (first 1500 chars): {full_text[:1500]}")
            else:
                print("WARNING: No text extracted after filtering!")
                print(f"Total OCR results: {len(results)}")
                if len(results) > 0:
                    print("Sample OCR results (first 10):")
                    for i, r in enumerate(results[:10]):
                        if len(r) > 1:
                            print(f"  {i+1}. Text: '{r[1][:100]}', Confidence: {r[2] if len(r) > 2 else 'N/A'}")
                
                # Если ничего не извлекли после фильтрации, пробуем с еще более низким порогом
                if len(results) > 0:
                    # Для изображений пробуем с очень низким порогом
                    lower_threshold = 0.05 if is_image_file else 0.1
                    print(f"Trying with lower confidence threshold ({lower_threshold})...")
                    low_threshold_results = [r for r in results if len(r) > 2 and r[2] > lower_threshold]
                    if len(low_threshold_results) > 0:
                        full_text = ' '.join([result[1] for result in low_threshold_results if len(result) > 1])
                        print(f"Text with lower threshold ({lower_threshold}): {len(full_text)} characters")
                        if len(full_text) > 0:
                            print(f"Sample text: {full_text[:500]}")
                    else:
                        # Последняя попытка - берем все результаты без фильтрации по уверенности
                        print("Trying without confidence filter...")
                        full_text = ' '.join([result[1] for result in results if len(result) > 1])
                        print(f"Text without filter: {len(full_text)} characters")
                        if len(full_text) > 0:
                            print(f"Sample text: {full_text[:500]}")
                    
                    if len(full_text) == 0:
                        print("ERROR: Still no text extracted after all attempts!")
                        print("This might indicate:")
                        print("  1. Image quality is too poor")
                        print("  2. Image doesn't contain text")
                        print("  3. OCR model failed to process the image")
                        print(f"  4. Processed file path: {processed_filepath}")
                        print(f"  5. Processed file exists: {os.path.exists(processed_filepath)}")
                        if os.path.exists(processed_filepath):
                            print(f"  6. Processed file size: {os.path.getsize(processed_filepath)} bytes")
                else:
                    print("ERROR: OCR returned no results at all!")
                    print("This might indicate:")
                    print("  1. Image is corrupted or unreadable")
                    print("  2. OCR model failed to initialize")
                    print(f"  3. Processed file path: {processed_filepath}")
                    print(f"  4. Processed file exists: {os.path.exists(processed_filepath)}")
            
            # Удаляем временный обработанный файл ПОСЛЕ извлечения текста
            if processed_filepath != filepath and os.path.exists(processed_filepath):
                try:
                    os.remove(processed_filepath)
                    print(f"Temporary processed file removed: {processed_filepath}")
                except Exception as e:
                    print(f"Warning: Failed to remove temporary file {processed_filepath}: {e}")
                    pass  # Игнорируем ошибки удаления
            
            # Удаляем конвертированный PDF файл, если он был создан
            if converted_pdf_path and os.path.exists(converted_pdf_path):
                try:
                    os.remove(converted_pdf_path)
                    print(f"Converted PDF file removed: {converted_pdf_path}")
                except Exception as e:
                    print(f"Warning: Failed to remove converted PDF file {converted_pdf_path}: {e}")
                    pass  # Игнорируем ошибки удаления

            # Автоматическое определение типа банка, если не указан
            detected_bank = self._extract_bank_type(full_text)
            if not bank_type and detected_bank:
                bank_type = detected_bank
            elif not bank_type:
                bank_type = 'Halyk'  # По умолчанию
            
            # Извлечение данных в зависимости от типа банка
            if bank_type == 'Halyk':
                result = self._extract_halyk_data(full_text, auto_extract_fullname)
            elif bank_type == 'Kaspi':
                result = self._extract_kaspi_data(full_text, auto_extract_fullname)
            else:
                result = self._extract_generic_data(full_text, auto_extract_fullname)
            
            # Добавляем определенный тип банка в результат
            result['bank_type'] = bank_type
            if detected_bank:
                result['detected_bank'] = detected_bank
            
            # Логируем извлеченные данные для отладки
            full_name = result.get('full_name') or ''
            full_name_display = full_name[:50] if full_name else 'N/A'
            print(f"Extracted data: amount={result.get('amount', 0)}, date={result.get('payment_date', 'N/A')}, name={full_name_display}")
            
            return result

        except Exception as e:
            import traceback
            error_trace = traceback.format_exc()
            print(f"Error in process_receipt: {e}")
            print(f"Traceback: {error_trace}")
            return {
                'full_name': '',
                'amount': 0.0,
                'payment_date': datetime.now().strftime('%Y-%m-%d'),
                'university_name': '',
                'error': str(e),
                'traceback': error_trace
            }

    def _extract_bank_type(self, text):
        """Автоматическое определение типа банка из текста квитанции"""
        text_lower = text.lower()
        text_original = text
        
        # Подсчитываем "вес" каждого банка на основе найденных маркеров
        kaspi_score = 0
        halyk_score = 0
        
        # Специальные маркеры для Kaspi Bank (высокий приоритет)
        if re.search(r'образование', text_lower):
            kaspi_score += 3
        if re.search(r'вузы\s+и\s+колледжи', text_lower):
            kaspi_score += 3
        if re.search(r'каспи\s+банк', text_lower):
            kaspi_score += 5
        if re.search(r'kaspi\s+bank', text_lower):
            kaspi_score += 5
        if re.search(r'каспи', text_lower):
            kaspi_score += 2
        if re.search(r'kaspi', text_lower):
            kaspi_score += 2
        
        # Маркеры для Halyk Bank
        if re.search(r'халык\s+банк', text_lower):
            halyk_score += 5
        if re.search(r'halyk\s+bank', text_lower):
            halyk_score += 5
        if re.search(r'халык', text_lower):
            halyk_score += 2
        if re.search(r'halyk', text_lower):
            halyk_score += 2
        
        print(f"Bank detection scores: Kaspi={kaspi_score}, Halyk={halyk_score}")
        
        # Определяем банк по максимальному весу
        if kaspi_score > halyk_score and kaspi_score > 0:
            print(f"Bank type detected as Kaspi (score: {kaspi_score})")
            return 'Kaspi'
        elif halyk_score > kaspi_score and halyk_score > 0:
            print(f"Bank type detected as Halyk (score: {halyk_score})")
            return 'Halyk'
        elif kaspi_score > 0:
            print(f"Bank type detected as Kaspi (score: {kaspi_score})")
            return 'Kaspi'
        elif halyk_score > 0:
            print(f"Bank type detected as Halyk (score: {halyk_score})")
            return 'Halyk'
        
        print("Bank type not detected in text")
        return None
    
    def _extract_halyk_data(self, text, auto_extract_fullname=False):
        """Извлечение данных из квитанции Halyk Bank"""
        full_name = self._extract_name(text, 'Halyk') if auto_extract_fullname else ''
        data = {
            'full_name': full_name or '',  # Гарантируем, что это строка, а не None
            'amount': self._extract_amount(text),
            'payment_date': self._extract_date(text),
            'university_name': self._extract_university(text)
        }
        return data

    def _extract_kaspi_data(self, text, auto_extract_fullname=False):
        """Извлечение данных из квитанции Kaspi Bank"""
        full_name = self._extract_name(text, 'Kaspi') if auto_extract_fullname else ''
        data = {
            'full_name': full_name or '',  # Гарантируем, что это строка, а не None
            'amount': self._extract_amount(text),
            'payment_date': self._extract_date(text),
            'university_name': self._extract_university(text)
        }
        return data

    def _extract_generic_data(self, text, auto_extract_fullname=False):
        """Универсальное извлечение данных"""
        full_name = self._extract_name(text, None) if auto_extract_fullname else ''
        data = {
            'full_name': full_name or '',  # Гарантируем, что это строка, а не None
            'amount': self._extract_amount(text),
            'payment_date': self._extract_date(text),
            'university_name': self._extract_university(text)
        }
        return data

    def _extract_name(self, text, bank_type=None):
        """Извлечение ФИО с учетом типа банка"""
        print(f"Extracting name from text (length: {len(text)}), bank_type: {bank_type}")
        print(f"First 1500 chars: {text[:1500]}")
        
        # Список слов, которые не должны быть частью ФИО
        exclude_words = [
            # Банки и финансы
            'банк', 'банка', 'банку', 'банком', 'банке', 'bank', 'тг', 'тенге', 'сумма', 
            'дата', 'квитанция', 'платеж', 'оплата', 'получатель', 'плательщик',
            'номер', 'реквизиты', 'счет', 'бик', 'иин', 'бин', 'кбе', 'кбе:', 'иин:',
            'год', 'месяц', 'день', 'время', 'часы', 'минуты', 'каспи', 'kaspi',
            'образование', 'вузы', 'колледжи', 'колледж', 'халык', 'halyk',
            # Географические названия и служебные слова
            'регион', 'область', 'город', 'г.', 'улица', 'ул.', 'проспект', 'пр.',
            'название', 'наззание', 'названи', 'названиe', 'названиe', 'названиe',
            'караганда', 'караандинский', 'карагандинский', 'алматы', 'астана', 'астане', 'павлодар', 'шымкент',
            'университет', 'университета', 'университете', 'университет', 'вуз', 'вуза', 'вузы',
            'экономический', 'экномический', 'экномиесский', 'экномиески', 'экономический',
            'факультет', 'курс', 'группа', 'студент', 'студента', 'студентки',
            'идентификатор', 'идентификатора', 'квитанции', 'квитанция',
            'казахстан', 'казахстана', 'казахстане', 'казахстанский',
            # Другие служебные слова
            'успешно', 'совершен', 'совершено', 'платеж', 'платежа', 'платежу',
            'дата', 'время', 'по', 'астане', 'астаны'
        ]
        
        found_candidates = []  # Список кандидатов на ФИО с приоритетами
        
        # Специальная обработка для Kaspi Bank
        if bank_type == 'Kaspi' or re.search(r'образование|вузы\s+и\s+колледжи', text, re.IGNORECASE):
            print("Processing as Kaspi Bank receipt, looking for FIO after 'Вузы и колледжи'...")
            
            # Ищем "Вузы и колледжи" и берем следующие три слова (ФИО идет сразу после)
            universities_match = re.search(r'вузы\s+и\s+колледжи', text, re.IGNORECASE)
            if universities_match:
                start_pos = universities_match.end()
                # Берем текст после маркера (до 150 символов - ФИО должно быть сразу, до других полей)
                text_after = text[start_pos:start_pos+150]
                print(f"Text after 'Вузы и колледжи' (first 150 chars): '{text_after}'")
                
                # Убираем лишние пробелы и переносы строк в начале
                text_after = text_after.strip()
                
                # Разбиваем текст на строки и ищем ФИО в первой непустой строке
                lines = text_after.split('\n')
                for line_idx, line in enumerate(lines[:2]):  # Проверяем только первые 2 строки (ФИО должно быть в первой)
                    line = line.strip()
                    if not line:
                        continue
                    
                    print(f"Checking line {line_idx} after 'Вузы и колледжи': '{line}'")
                    
                    # Ищем три слова с заглавными буквами в строке
                    # Паттерн: три слова, каждое начинается с заглавной, минимум 4 буквы
                    # Используем более гибкий паттерн для учета разных вариантов написания
                    fio_pattern = r'^([А-ЯЁ][а-яё]{4,}(?:-[А-ЯЁ][а-яё]{2,})?)\s+([А-ЯЁ][а-яё]{4,})\s+([А-ЯЁ][а-яё]{4,})'
                    match = re.match(fio_pattern, line)
                    if match:
                        name = f"{match.group(1)} {match.group(2)} {match.group(3)}"
                        name = name.strip()
                        words = name.split()
                        if len(words) == 3:
                            # Проверяем, что не содержит исключенных слов
                            name_lower = name.lower()
                            words_lower = [w.lower() for w in words]
                            
                            # Проверяем, что ни одно слово не является исключенным (регистронезависимо)
                            if not any(ex_word in name_lower for ex_word in exclude_words):
                                # Проверяем, что отдельные слова не являются исключенными
                                if not any(word_lower in exclude_words for word_lower in words_lower):
                                    # Проверяем, что все слова начинаются с заглавной и достаточно длинные (минимум 4 буквы)
                                    if all(word and word[0].isupper() and len(word) >= 4 for word in words):
                                        # Проверяем наличие кириллицы
                                        if re.search(r'[А-ЯЁа-яё]', name):
                                            # Проверяем, что слова не содержат слишком много заглавных букв (имена обычно только с первой заглавной)
                                            # Исключаем слова, где больше 30% заглавных букв (кроме первой)
                                            def has_too_many_uppercase(word):
                                                if len(word) <= 1:
                                                    return False
                                                rest = word[1:]
                                                if len(rest) == 0:
                                                    return False
                                                uppercase_count = sum(1 for c in rest if c.isupper())
                                                return uppercase_count / len(rest) > 0.3
                                            
                                            if not any(has_too_many_uppercase(word) for word in words):
                                                # Дополнительная проверка: слова не должны быть географическими названиями или названиями учреждений
                                                geographic_words = ['караганда', 'караандинский', 'карагандинский', 'алматы', 'астана', 
                                                                   'павлодар', 'шымкент', 'регион', 'область', 'город', 'улица', 
                                                                   'проспект', 'университет', 'вуз', 'вуза', 'экономический', 
                                                                   'экномический', 'экномиесский', 'название', 'наззание']
                                                if not any(word_lower in geographic_words for word_lower in words_lower):
                                                    found_candidates.append((name, 10, line_idx))  # Высокий приоритет
                                                    print(f"✓ Found Kaspi FIO in line {line_idx} after 'Вузы и колледжи': {name}")
                                                    break
                
                # Если не нашли в строках, пробуем найти три слова сразу после маркера (без разбиения на строки)
                if not found_candidates or (found_candidates and found_candidates[-1][1] != 10):
                    # Берем первые 100 символов после маркера
                    short_text = text_after[:100].strip()
                    print(f"Trying to find FIO in first 100 chars without line breaks: '{short_text}'")
                    
                    # Ищем три слова с заглавными буквами
                    fio_pattern = r'^([А-ЯЁ][а-яё]{4,}(?:-[А-ЯЁ][а-яё]{2,})?)\s+([А-ЯЁ][а-яё]{4,})\s+([А-ЯЁ][а-яё]{4,})'
                    match = re.match(fio_pattern, short_text)
                    if match:
                        name = f"{match.group(1)} {match.group(2)} {match.group(3)}"
                        name = name.strip()
                        words = name.split()
                        if len(words) == 3:
                            name_lower = name.lower()
                            words_lower = [w.lower() for w in words]
                            
                            if not any(ex_word in name_lower for ex_word in exclude_words):
                                if not any(word_lower in exclude_words for word_lower in words_lower):
                                    if all(word and word[0].isupper() and len(word) >= 4 for word in words):
                                        if re.search(r'[А-ЯЁа-яё]', name):
                                            def has_too_many_uppercase(word):
                                                if len(word) <= 1:
                                                    return False
                                                rest = word[1:]
                                                if len(rest) == 0:
                                                    return False
                                                uppercase_count = sum(1 for c in rest if c.isupper())
                                                return uppercase_count / len(rest) > 0.3
                                            
                                            if not any(has_too_many_uppercase(word) for word in words):
                                                geographic_words = ['караганда', 'караандинский', 'карагандинский', 'алматы', 'астана', 
                                                                   'павлодар', 'шымкент', 'регион', 'область', 'город', 'улица', 
                                                                   'проспект', 'университет', 'вуз', 'вуза', 'экономический', 
                                                                   'экномический', 'экномиесский', 'название', 'наззание']
                                                if not any(word_lower in geographic_words for word_lower in words_lower):
                                                    found_candidates.append((name, 10, 0))
                                                    print(f"✓ Found Kaspi FIO immediately after 'Вузы и колледжи': {name}")
                
                # Если не нашли в строках, пробуем паттерн без разбиения на строки
                if not found_candidates or found_candidates[-1][1] != 10:
                    # Ищем три слова с заглавными буквами сразу после маркера
                    fio_pattern = r'^[^\w]*([А-ЯЁ][а-яё]{3,}(?:-[А-ЯЁ][а-яё]{2,})?)\s+([А-ЯЁ][а-яё]{3,})\s+([А-ЯЁ][а-яё]{3,})'
                    match = re.search(fio_pattern, text_after, re.MULTILINE)
                    if match:
                        name = f"{match.group(1)} {match.group(2)} {match.group(3)}"
                        name = name.strip()
                        words = name.split()
                        if len(words) == 3:
                            name_lower = name.lower()
                            if not any(ex_word in name_lower for ex_word in exclude_words):
                                if all(word and word[0].isupper() and len(word) >= 3 for word in words):
                                    if re.search(r'[А-ЯЁа-яё]', name):
                                        if not any(word.lower() in ['караганда', 'алматы', 'астана', 'павлодар', 'шымкент', 
                                                                     'регион', 'область', 'город', 'улица', 'проспект'] 
                                                   for word in words):
                                            found_candidates.append((name, 10, 0))
                                            print(f"Found Kaspi FIO immediately after 'Вузы и колледжи': {name}")
                
                # Если не нашли с якорями, пробуем без якорей (но в первых 100 символах)
                if not found_candidates or found_candidates[-1][1] != 10:
                    short_text = text_after[:100]  # Берем только первые 100 символов
                    fio_pattern = r'([А-ЯЁ][а-яё]{2,}(?:-[А-ЯЁ][а-яё]{2,})?\s+[А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,})'
                    matches = list(re.finditer(fio_pattern, short_text))
                    for match in matches[:1]:  # Берем только первое совпадение
                        name = match.group(1).strip()
                        words = name.split()
                        if len(words) == 3:
                            name_lower = name.lower()
                            if not any(ex_word in name_lower for ex_word in exclude_words):
                                if all(word and word[0].isupper() and len(word) >= 2 for word in words):
                                    if re.search(r'[А-ЯЁа-яё]', name):
                                        found_candidates.append((name, 10, match.start()))
                                        print(f"Found Kaspi FIO in first 100 chars after 'Вузы и колледжи': {name}")
                                        break
        
        # Стандартные паттерны для поиска ФИО (с приоритетами)
        standard_patterns = [
            (r'ФИО[:\s\-]+([А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,})', 8),
            (r'Плательщик[:\s\-]+([А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,})', 7),
            (r'Получатель[:\s\-]+([А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,})', 7),
            (r'Ф\.?И\.?О\.?[:\s\-]+([А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ][а-яё]{2,})', 8),
            (r'([А-ЯЁ][а-яё]{2,}\s+[А-ЯЁ]\.\s*[А-ЯЁ]\.)', 6),  # С инициалами
            (r'\b([А-ЯЁ][а-яё]{3,}\s+[А-ЯЁ][а-яё]{3,}\s+[А-ЯЁ][а-яё]{3,})\b', 5),  # Три слова
        ]
        
        for pattern, priority in standard_patterns:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                name = match.group(1).strip()
                name = re.sub(r'[^\w\s\-]', ' ', name)
                name = ' '.join(name.split())
                words = name.split()
                
                if len(words) == 3 and len(name) > 8:
                    if not re.match(r'^\d', name):
                        name_lower = name.lower()
                        if not any(ex_word in name_lower for ex_word in exclude_words):
                            if re.search(r'[А-ЯЁа-яё]', name):
                                if all(word and word[0].isupper() and len(word) >= 3 for word in words):
                                    found_candidates.append((name, priority, match.start()))
                                    print(f"Found FIO with pattern (priority {priority}): {name}")
        
        # Если нашли кандидатов, выбираем лучшего (по приоритету, затем по позиции)
        if found_candidates:
            found_candidates.sort(key=lambda x: (-x[1], x[2]))  # Сортируем по приоритету (убывание), затем по позиции
            selected_name = found_candidates[0][0]
            print(f"Selected FIO (priority {found_candidates[0][1]}): {selected_name}")
            return selected_name
        
        # Последняя попытка: ищем любые три слова с заглавными
        words = text.split()
        for i in range(len(words) - 2):
            word1, word2, word3 = words[i], words[i+1], words[i+2]
            if (word1 and word1[0].isupper() and len(word1) >= 3 and
                word2 and word2[0].isupper() and len(word2) >= 3 and
                word3 and word3[0].isupper() and len(word3) >= 3):
                if (not re.match(r'^\d', word1) and not re.match(r'^\d', word2) and not re.match(r'^\d', word3)):
                    if (word1.lower() not in exclude_words and 
                        word2.lower() not in exclude_words and 
                        word3.lower() not in exclude_words):
                        if (re.search(r'[А-ЯЁа-яё]', word1) or 
                            re.search(r'[А-ЯЁа-яё]', word2) or 
                            re.search(r'[А-ЯЁа-яё]', word3)):
                            name = f"{word1} {word2} {word3}"
                            print(f"Found FIO by word sequence: {name}")
                            return name
        
        print("No FIO found in text")
        return ''

    def _extract_amount(self, text):
        """Извлечение суммы"""
        print(f"Extracting amount from text (length: {len(text)})")
        print(f"First 1000 chars of text: {text[:1000]}")
        
        # Расширенные паттерны для поиска суммы
        patterns = [
            # С явным указанием "Сумма"
            r'Сумма[:\s]+(\d{1,3}(?:\s?\d{3})*[.,]\d{2})',
            r'Сумма[:\s]+(\d{1,3}(?:\s?\d{3})*[.,]?\d{0,2})',
            r'Сумма к оплате[:\s]+(\d{1,3}(?:\s?\d{3})*[.,]\d{2})',
            r'Сумма к оплате[:\s]+(\d{1,3}(?:\s?\d{3})*[.,]?\d{0,2})',
            r'Сумма платежа[:\s]+(\d{1,3}(?:\s?\d{3})*[.,]\d{2})',
            r'К оплате[:\s]+(\d{1,3}(?:\s?\d{3})*[.,]\d{2})',
            # С валютой
            r'(\d{1,3}(?:\s?\d{3})*[.,]\d{2})\s*тг',
            r'(\d{1,3}(?:\s?\d{3})*[.,]\d{2})\s*тенге',
            r'(\d{1,3}(?:\s?\d{3})*[.,]\d{2})\s*₸',
            r'(\d{1,3}(?:\s?\d{3})*[.,]\d{2})\s*KZT',
            # Без валюты, но с десятичными знаками
            r'(\d{1,3}(?:\s?\d{3})*[.,]\d{2})',
            # Без десятичных знаков (целые числа)
            r'Сумма[:\s]+(\d{1,3}(?:\s?\d{3})+)',
            r'К оплате[:\s]+(\d{1,3}(?:\s?\d{3})+)',
            r'(\d{4,})\s*тг',
            r'(\d{4,})\s*тенге',
            r'(\d{4,})\s*₸',
            # Числа с пробелами как разделители тысяч
            r'(\d{1,3}(?:\s\d{3})+[.,]?\d{0,2})',
            # Числа без пробелов (большие суммы)
            r'(\d{5,}[.,]?\d{0,2})',
        ]

        amounts = []
        for i, pattern in enumerate(patterns):
            matches = re.findall(pattern, text, re.IGNORECASE)
            for match in matches:
                try:
                    # Замена запятой на точку и удаление пробелов
                    amount_str = str(match).replace(',', '.').replace(' ', '').strip()
                    # Удаляем все нецифровые символы кроме точки
                    amount_str = re.sub(r'[^\d.]', '', amount_str)
                    
                    if amount_str and amount_str.replace('.', '').isdigit():
                        amount = float(amount_str)
                        # Фильтруем слишком маленькие суммы (меньше 1000) и слишком большие (больше 10 миллионов)
                        if 1000 <= amount <= 10000000:
                            amounts.append(amount)
                            print(f"Pattern {i+1} found amount: {amount} (from '{match}')")
                except Exception as e:
                    print(f"Error parsing amount '{match}': {e}")
                    continue

        # Если не нашли с паттернами, пробуем найти любые большие числа
        if not amounts:
            print("No amounts found with patterns, trying to find large numbers...")
            # Ищем все числа в тексте
            all_numbers = re.findall(r'\d{4,}[.,]?\d{0,2}', text)
            for num_str in all_numbers:
                try:
                    amount_str = num_str.replace(',', '.').replace(' ', '').strip()
                    amount_str = re.sub(r'[^\d.]', '', amount_str)
                    if amount_str and amount_str.replace('.', '').isdigit():
                        amount = float(amount_str)
                        if 1000 <= amount <= 10000000:
                            amounts.append(amount)
                            print(f"Found large number: {amount} (from '{num_str}')")
                except:
                    continue

        print(f"All found amounts: {amounts}")
        
        # Возвращаем наибольшую сумму (обычно это сумма платежа)
        result = max(amounts) if amounts else 0.0
        print(f"Final extracted amount: {result}")
        return result

    def _extract_date(self, text):
        """Извлечение даты платежа"""
        # Паттерны для поиска даты
        patterns = [
            r'(\d{2}[./]\d{2}[./]\d{4})',
            r'(\d{4}[./-]\d{2}[./-]\d{2})',
            r'Дата[:\s]+(\d{2}[./]\d{2}[./]\d{4})',
            r'Дата платежа[:\s]+(\d{2}[./]\d{2}[./]\d{4})',
        ]

        for pattern in patterns:
            match = re.search(pattern, text)
            if match:
                date_str = match.group(1)
                # Попытка распарсить дату
                for fmt in ['%d.%m.%Y', '%d/%m/%Y', '%Y-%m-%d', '%d.%m.%y']:
                    try:
                        date = datetime.strptime(date_str.replace('/', '.'), fmt)
                        if date.year < 2000:
                            date = date.replace(year=date.year + 2000)
                        return date.strftime('%Y-%m-%d')
                    except:
                        continue

        # Если дата не найдена, возвращаем текущую дату
        return datetime.now().strftime('%Y-%m-%d')

    def _extract_university(self, text):
        """Извлечение наименования ВУЗа"""
        # Паттерны для поиска ВУЗа
        patterns = [
            r'Университет[:\s]+([А-ЯЁ][А-ЯЁа-яё\s]+)',
            r'ВУЗ[:\s]+([А-ЯЁ][А-ЯЁа-яё\s]+)',
            r'Получатель[:\s]+([А-ЯЁ][А-ЯЁа-яё\s]+(?:университет|институт|академия))',
        ]

        for pattern in patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                return match.group(1).strip()

        # Поиск по ключевым словам
        keywords = ['университет', 'институт', 'академия', 'колледж']
        words = text.split()
        for i, word in enumerate(words):
            if any(kw in word.lower() for kw in keywords):
                # Берем несколько слов вокруг ключевого слова
                start = max(0, i - 2)
                end = min(len(words), i + 5)
                return ' '.join(words[start:end])

        return ''

