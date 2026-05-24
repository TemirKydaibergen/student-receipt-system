// Глобальные переменные
let authToken = localStorage.getItem('authToken');
let currentUser = JSON.parse(localStorage.getItem('currentUser') || '{}');
let groups = [];
let currentPage = 1;
let totalPages = 1;
const itemsPerPage = 20; // Количество квитанций на странице
let searchTimeout; // Таймер для debounce поиска

// Универсальная функция для открытия модального окна с поддержкой вложенности
function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    
    // Проверяем, есть ли открытые модальные окна (кроме текущего)
    const openModals = document.querySelectorAll('.modal.show');
    if (openModals.length > 0) {
        // Если есть открытые модальные окна, делаем это вложенным
        modal.classList.add('modal-nested');
    } else {
        // Если нет открытых модальных окон, убираем класс вложенности
        modal.classList.remove('modal-nested');
    }
    
    modal.style.display = 'flex';
    setTimeout(() => modal.classList.add('show'), 10);
}

// Универсальная функция для закрытия модального окна
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (!modal) return;
    
    modal.style.display = 'none';
    modal.classList.remove('show');
    modal.classList.remove('modal-nested');
}

// Инициализация
document.addEventListener('DOMContentLoaded', () => {
    checkAuth();
    loadGroups();
    loadBanks();
    loadBanksForFilter();
    setupEventListeners();
});

// Проверка авторизации
function checkAuth() {
    if (authToken && currentUser.username) {
        showMainSection();
    } else {
        showAuthSection();
    }
}

// Показать секцию авторизации
function showAuthSection() {
    document.getElementById('auth-section').style.display = 'flex';
    document.getElementById('main-section').style.display = 'none';
    document.getElementById('logout-btn').style.display = 'none';
}

// Показать основную секцию
function showMainSection() {
    document.getElementById('auth-section').style.display = 'none';
    document.getElementById('main-section').style.display = 'block';
    document.getElementById('username-display').textContent = currentUser.username;
    document.getElementById('logout-btn').style.display = 'inline-block';
    
    // Показать кнопки в header
    const headerActions = document.getElementById('header-actions');
    if (headerActions) {
        headerActions.style.display = 'flex';
    }
    
    // Показать админские кнопки
    if (currentUser.role === 'admin') {
        const adminSettingsBtn = document.getElementById('admin-settings-btn');
        const adminReportsBtn = document.getElementById('admin-reports-btn');
        
        if (adminSettingsBtn) adminSettingsBtn.style.display = 'inline-flex';
        if (adminReportsBtn) adminReportsBtn.style.display = 'inline-flex';
    }
    
    loadReceipts();
}

// Настройка обработчиков событий
function setupEventListeners() {
    // Авторизация
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('logout-btn').addEventListener('click', handleLogout);
    
    // Пагинация
    const prevPageBtn = document.getElementById('prev-page');
    const nextPageBtn = document.getElementById('next-page');
    
    if (prevPageBtn) {
        prevPageBtn.addEventListener('click', () => {
            if (currentPage > 1) {
                currentPage--;
                loadReceipts();
            }
        });
    }
    if (nextPageBtn) {
        nextPageBtn.addEventListener('click', () => {
            currentPage++;
            loadReceipts();
        });
    }

    // Загрузка квитанций
    const uploadBtn = document.getElementById('upload-btn');
    const uploadForm = document.getElementById('upload-form');
    const cancelUploadBtn = document.getElementById('cancel-upload');
    
    if (uploadBtn) {
        uploadBtn.addEventListener('click', () => {
            const modal = document.getElementById('upload-modal');
            if (modal) {
                // Убеждаемся, что чекбоксы включены при открытии модального окна
                const autoExtractFullname = document.getElementById('auto-extract-fullname');
                const autoExtractBank = document.getElementById('auto-extract-bank');
                if (autoExtractFullname) {
                    autoExtractFullname.checked = true;
                    // Триггерим событие change для применения состояния
                    autoExtractFullname.dispatchEvent(new Event('change'));
                }
                if (autoExtractBank) {
                    autoExtractBank.checked = true;
                    // Триггерим событие change для применения состояния
                    autoExtractBank.dispatchEvent(new Event('change'));
                }
                modal.style.display = 'flex';
                modal.classList.add('show');
            }
        });
    }
    if (uploadForm) {
        uploadForm.addEventListener('submit', handleUpload);
    }
    if (cancelUploadBtn) {
        cancelUploadBtn.addEventListener('click', () => {
            const modal = document.getElementById('upload-modal');
            if (modal) {
                modal.style.display = 'none';
                modal.classList.remove('show');
            }
            if (uploadForm) {
                uploadForm.reset();
                // После сброса формы снова включаем чекбоксы
                const autoExtractFullname = document.getElementById('auto-extract-fullname');
                const autoExtractBank = document.getElementById('auto-extract-bank');
                if (autoExtractFullname) {
                    autoExtractFullname.checked = true;
                    autoExtractFullname.dispatchEvent(new Event('change'));
                }
                if (autoExtractBank) {
                    autoExtractBank.checked = true;
                    autoExtractBank.dispatchEvent(new Event('change'));
                }
            }
            const fileNameEl = document.getElementById('file-name');
            if (fileNameEl) {
                fileNameEl.style.display = 'none';
            }
        });
    }

    // Универсальный поиск
    const globalSearch = document.getElementById('global-search');
    const clearSearchBtn = document.getElementById('clear-search');
    
    if (globalSearch) {
        globalSearch.addEventListener('input', (e) => {
            clearTimeout(searchTimeout);
            const value = e.target.value.trim();
            
            if (value) {
                if (clearSearchBtn) clearSearchBtn.style.display = 'flex';
            } else {
                if (clearSearchBtn) clearSearchBtn.style.display = 'none';
            }
            
            // Поиск с задержкой (debounce)
            searchTimeout = setTimeout(() => {
                currentPage = 1; // Сбрасываем на первую страницу при поиске
                loadReceipts();
            }, 500);
        });
        
        globalSearch.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                clearTimeout(searchTimeout);
                const value = globalSearch.value.trim();
                console.log('Search triggered (Enter) with value:', value);
                currentPage = 1; // Сбрасываем на первую страницу при поиске
                loadReceipts();
            }
        });
    }
    
    if (clearSearchBtn) {
        clearSearchBtn.addEventListener('click', () => {
            if (globalSearch) {
                globalSearch.value = '';
                clearSearchBtn.style.display = 'none';
                console.log('Search cleared');
                currentPage = 1; // Сбрасываем на первую страницу
                loadReceipts();
            }
        });
    }
    
    // Фильтры - добавляем обработчики на изменение фильтров
    const filterGroup = document.getElementById('filter-group');
    const filterBank = document.getElementById('filter-bank');
    const filterStatus = document.getElementById('filter-status');
    const filterDateFrom = document.getElementById('filter-date-from');
    const filterDateTo = document.getElementById('filter-date-to');
    
    if (filterGroup) {
        filterGroup.addEventListener('change', (e) => {
            const value = e.target.value;
            console.log('Filter group changed:', value);
            currentPage = 1;
            loadReceipts();
        });
    }
    if (filterBank) {
        filterBank.addEventListener('change', () => {
            console.log('Filter bank changed:', filterBank.value);
            currentPage = 1;
            loadReceipts();
        });
    }
    if (filterStatus) {
        filterStatus.addEventListener('change', () => {
            console.log('Filter status changed:', filterStatus.value);
            currentPage = 1;
            loadReceipts();
        });
    }
    if (filterDateFrom) {
        filterDateFrom.addEventListener('change', () => {
            console.log('Filter date from changed:', filterDateFrom.value);
            currentPage = 1;
            loadReceipts();
        });
    }
    if (filterDateTo) {
        filterDateTo.addEventListener('change', () => {
            console.log('Filter date to changed:', filterDateTo.value);
            currentPage = 1;
            loadReceipts();
        });
    }
    
    // Кнопка очистки фильтров
    const clearFiltersBtn = document.getElementById('clear-filters');
    if (clearFiltersBtn) {
        clearFiltersBtn.addEventListener('click', () => {
            console.log('Clear filters button clicked');
            clearAllFilters();
        });
    }
    
    document.getElementById('refresh-btn').addEventListener('click', () => {
        currentPage = 1;
        loadReceipts();
    });
    
    // Автоматическая загрузка банков в фильтр
    loadBanksForFilter();

    // Единое модальное окно настроек
    const adminSettingsBtn = document.getElementById('admin-settings-btn');
    if (adminSettingsBtn) {
        adminSettingsBtn.addEventListener('click', () => {
            const modal = document.getElementById('settings-modal');
            if (modal) {
                modal.style.display = 'flex';
                modal.classList.add('show');
                // Загружаем данные для активного таба
                switchToTab('users');
            }
        });
    }
    
    // Модальное окно отчетов
    const adminReportsBtn = document.getElementById('admin-reports-btn');
    if (adminReportsBtn) {
        adminReportsBtn.addEventListener('click', () => {
            const modal = document.getElementById('reports-modal');
            if (modal) {
                modal.style.display = 'flex';
                modal.classList.add('show');
                loadReportYears();
                loadReportGroups();
            }
        });
    }
    
    // Обработчики для отчетов
    const loadReportBtn = document.getElementById('load-report-btn');
    if (loadReportBtn) {
        loadReportBtn.addEventListener('click', loadGroupReport);
    }
    
    const exportReportBtn = document.getElementById('export-report-btn');
    if (exportReportBtn) {
        exportReportBtn.addEventListener('click', exportGroupReport);
    }
    
    // Экспорт всех квитанций (в модальном окне отчетов)
    const exportAllBtn = document.getElementById('export-all-btn');
    if (exportAllBtn) {
        exportAllBtn.addEventListener('click', handleExport);
    }
    
    // Импорт студентов
    const importBtn = document.getElementById('import-btn');
    if (importBtn) {
        importBtn.addEventListener('click', () => {
            const modal = document.getElementById('import-modal');
            if (modal) {
                modal.style.display = 'flex';
                modal.classList.add('show');
            }
        });
    }
    
    const importFileUploadArea = document.getElementById('import-file-upload-area');
    const importFileInput = document.getElementById('import-file');
    const importFileName = document.getElementById('import-file-name');
    const importSubmitBtn = document.getElementById('import-submit-btn');
    
    if (importFileUploadArea && importFileInput) {
        importFileUploadArea.addEventListener('click', () => importFileInput.click());
        
        importFileInput.addEventListener('change', async (e) => {
            if (e.target.files.length > 0) {
                const file = e.target.files[0];
                importFileName.textContent = `📄 ${file.name}`;
                importFileName.style.display = 'block';
                importFileUploadArea.style.borderColor = '#28a745';
                importFileUploadArea.style.background = '#d4edda';
                
                // Парсим Excel файл
                try {
                    await previewImportFile(file);
                    if (importSubmitBtn) importSubmitBtn.style.display = 'inline-block';
                } catch (error) {
                    console.error('Error previewing file:', error);
                    showNotification('❌', 'Ошибка чтения файла: ' + error.message, 'error');
                }
            }
        });
    }
    
    if (importSubmitBtn) {
        importSubmitBtn.addEventListener('click', handleImportStudents);
    }
    
    // Скачивание шаблона для импорта
    const downloadTemplateBtn = document.getElementById('download-template-btn');
    if (downloadTemplateBtn) {
        downloadTemplateBtn.addEventListener('click', downloadImportTemplate);
    }
    
    // Переключение табов
    const settingsTabs = document.querySelectorAll('.settings-tab');
    settingsTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const tabName = tab.getAttribute('data-tab');
            switchToTab(tabName);
        });
    });
    
    // Добавление учебного года
    document.getElementById('add-academic-year-btn').addEventListener('click', () => {
        document.getElementById('edit-academic-year-id').value = '';
        document.getElementById('edit-academic-year-form').reset();
        document.getElementById('edit-academic-year-title').textContent = '📅 Добавить учебный год';
        openModal('edit-academic-year-modal');
    });
    
    // Форма редактирования учебного года
    document.getElementById('edit-academic-year-form').addEventListener('submit', handleSaveAcademicYear);
    
    // Форма установки требуемой суммы
    document.getElementById('set-group-amount-form').addEventListener('submit', handleSetGroupAmount);
    
    // Кнопка добавления требуемой суммы
    const addGroupAmountBtn = document.getElementById('add-group-amount-btn');
    if (addGroupAmountBtn) {
        addGroupAmountBtn.addEventListener('click', () => {
            document.getElementById('edit-group-amount-id').value = '';
            document.getElementById('set-group-amount-form').reset();
            document.getElementById('group-amount-form-title').textContent = '➕ Добавить требуемую сумму';
            document.getElementById('group-amount-form-container').style.display = 'block';
            document.getElementById('group-amount-form-container').scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        });
    }
    
    // Кнопка отмены формы
    const cancelGroupAmountBtn = document.getElementById('cancel-group-amount-btn');
    if (cancelGroupAmountBtn) {
        cancelGroupAmountBtn.addEventListener('click', () => {
            document.getElementById('set-group-amount-form').reset();
            document.getElementById('edit-group-amount-id').value = '';
            document.getElementById('group-amount-form-container').style.display = 'none';
        });
    }
    
    // Группы
    document.getElementById('add-group-btn').addEventListener('click', () => {
        document.getElementById('edit-group-id').value = '';
        document.getElementById('edit-group-name').value = '';
        document.getElementById('edit-group-title').textContent = '📚 Добавить группу';
        openModal('edit-group-modal');
    });
    document.getElementById('edit-group-form').addEventListener('submit', handleSaveGroup);
    document.getElementById('cancel-edit-group').addEventListener('click', () => {
        closeModal('edit-group-modal');
        document.getElementById('edit-group-form').reset();
    });
    
    // Банки
    document.getElementById('add-bank-btn').addEventListener('click', () => {
        document.getElementById('edit-bank-id').value = '';
        document.getElementById('edit-bank-name').value = '';
        document.getElementById('edit-bank-title').textContent = '🏦 Добавить банк';
        openModal('edit-bank-modal');
    });
    document.getElementById('edit-bank-form').addEventListener('submit', handleSaveBank);
    document.getElementById('cancel-edit-bank').addEventListener('click', () => {
        closeModal('edit-bank-modal');
        document.getElementById('edit-bank-form').reset();
    });
    
    // Галочка автоматического извлечения ФИО
    document.getElementById('auto-extract-fullname').addEventListener('change', (e) => {
        const fullnameInput = document.getElementById('upload-fullname');
        if (e.target.checked) {
            fullnameInput.removeAttribute('required');
            fullnameInput.disabled = true;
            fullnameInput.placeholder = 'Будет извлечено автоматически из квитанции';
        } else {
            fullnameInput.setAttribute('required', 'required');
            fullnameInput.disabled = false;
            fullnameInput.placeholder = 'Введите ФИО студента';
        }
    });
    
    // Галочка автоматического извлечения банка
    document.getElementById('auto-extract-bank').addEventListener('change', (e) => {
        const bankSelect = document.getElementById('upload-bank');
        if (e.target.checked) {
            bankSelect.removeAttribute('required');
            bankSelect.disabled = true;
            bankSelect.style.opacity = '0.6';
        } else {
            bankSelect.setAttribute('required', 'required');
            bankSelect.disabled = false;
            bankSelect.style.opacity = '1';
        }
    });
    document.getElementById('add-user-btn').addEventListener('click', () => {
        openModal('add-user-modal');
        document.getElementById('add-user-form').reset();
    });
    document.getElementById('add-user-form').addEventListener('submit', handleAddUser);
    document.getElementById('cancel-add-user').addEventListener('click', () => {
        closeModal('add-user-modal');
        document.getElementById('add-user-form').reset();
    });

    // Закрытие модальных окон
    document.querySelectorAll('.close').forEach(closeBtn => {
        closeBtn.addEventListener('click', (e) => {
            const modal = e.target.closest('.modal');
            if (modal && modal.id) {
                // Если закрывается модальное окно загрузки, сбрасываем форму и включаем чекбоксы
                if (modal.id === 'upload-modal') {
                    const uploadForm = document.getElementById('upload-form');
                    if (uploadForm) {
                        uploadForm.reset();
                        const autoExtractFullname = document.getElementById('auto-extract-fullname');
                        const autoExtractBank = document.getElementById('auto-extract-bank');
                        if (autoExtractFullname) {
                            autoExtractFullname.checked = true;
                            autoExtractFullname.dispatchEvent(new Event('change'));
                        }
                        if (autoExtractBank) {
                            autoExtractBank.checked = true;
                            autoExtractBank.dispatchEvent(new Event('change'));
                        }
                    }
                }
                closeModal(modal.id);
            }
        });
    });

    window.addEventListener('click', (e) => {
        if (e.target.classList.contains('modal') && e.target.id) {
            // Если закрывается модальное окно загрузки, сбрасываем форму и включаем чекбоксы
            if (e.target.id === 'upload-modal') {
                const uploadForm = document.getElementById('upload-form');
                if (uploadForm) {
                    uploadForm.reset();
                    const autoExtractFullname = document.getElementById('auto-extract-fullname');
                    const autoExtractBank = document.getElementById('auto-extract-bank');
                    if (autoExtractFullname) {
                        autoExtractFullname.checked = true;
                        autoExtractFullname.dispatchEvent(new Event('change'));
                    }
                    if (autoExtractBank) {
                        autoExtractBank.checked = true;
                        autoExtractBank.dispatchEvent(new Event('change'));
                    }
                }
            }
            closeModal(e.target.id);
        }
    });
    
    // Обработка загрузки файла
    const fileUploadArea = document.getElementById('file-upload-area');
    const fileInput = document.getElementById('upload-file');
    const fileName = document.getElementById('file-name');
    
    if (fileUploadArea && fileInput) {
        fileUploadArea.addEventListener('click', () => fileInput.click());
        
        fileInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                fileName.textContent = `📄 ${e.target.files[0].name}`;
                fileName.style.display = 'block';
                fileUploadArea.style.borderColor = '#28a745';
                fileUploadArea.style.background = '#d4edda';
            }
        });
        
        // Drag and drop
        fileUploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            fileUploadArea.classList.add('dragover');
        });
        
        fileUploadArea.addEventListener('dragleave', () => {
            fileUploadArea.classList.remove('dragover');
        });
        
        fileUploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            fileUploadArea.classList.remove('dragover');
            if (e.dataTransfer.files.length > 0) {
                fileInput.files = e.dataTransfer.files;
                fileName.textContent = `📄 ${e.dataTransfer.files[0].name}`;
                fileName.style.display = 'block';
                fileUploadArea.style.borderColor = '#28a745';
                fileUploadArea.style.background = '#d4edda';
            }
        });
    }

    // Обработчик кнопки переноса переплаты
    const transferSubmitBtn = document.getElementById('transfer-submit-btn');
    if (transferSubmitBtn) {
        transferSubmitBtn.addEventListener('click', handleTransferOverpayment);
    }

    // Обработчик закрытия модального окна переноса переплаты
    const transferModal = document.getElementById('transfer-overpayment-modal');
    if (transferModal) {
        const closeButtons = transferModal.querySelectorAll('.close');
        closeButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                transferModal.style.display = 'none';
                transferModal.classList.remove('show');
                document.getElementById('transfer-overpayment-form').reset();
            });
        });
    }
}

// API функции
async function apiCall(endpoint, options = {}) {
    const url = `/api${endpoint}`;
    const defaultOptions = {
        headers: {},
    };

    // Добавляем Content-Type только если это не FormData
    if (!(options.body instanceof FormData)) {
        defaultOptions.headers['Content-Type'] = 'application/json';
    }

    if (authToken) {
        defaultOptions.headers['Authorization'] = `Bearer ${authToken}`;
    }

    const finalOptions = { ...defaultOptions, ...options };
    
    try {
        const response = await fetch(url, finalOptions);
        
        // Проверяем, есть ли тело ответа
        const contentType = response.headers.get('content-type');
        const isJson = contentType && contentType.includes('application/json');
        
        let data = null;
        
        // Пытаемся прочитать тело ответа только если это JSON
        if (isJson) {
            const text = await response.text();
            if (text && text.trim()) {
                try {
                    data = JSON.parse(text);
                } catch (parseError) {
                    console.error('JSON parse error:', parseError, 'Response text:', text);
                    throw new Error('Неверный формат ответа от сервера');
                }
            }
        }
        
        if (!response.ok) {
            const errorMessage = data?.error || `Ошибка ${response.status}: ${response.statusText}`;
            throw new Error(errorMessage);
        }
        
        return data;
    } catch (error) {
        console.error('API Error:', error);
        if (error.message) {
            showNotification('❌', error.message, 'error');
        } else {
            showNotification('❌', 'Ошибка при выполнении запроса', 'error');
        }
        throw error;
    }
}

// Авторизация
async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    try {
        const response = await apiCall('/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
        });

        if (!response || !response.token) {
            showNotification('❌', 'Неверный ответ от сервера при авторизации', 'error');
            console.error('Invalid login response:', response);
            return;
        }

        authToken = response.token;
        currentUser = { username: response.username, role: response.role };
        
        localStorage.setItem('authToken', authToken);
        localStorage.setItem('currentUser', JSON.stringify(currentUser));
        
        showNotification('✅', 'Успешный вход в систему!', 'success');
        showMainSection();
    } catch (error) {
        console.error('Login error:', error);
        // Ошибка уже обработана в apiCall
    }
}


function handleLogout() {
    authToken = null;
    currentUser = {};
    localStorage.removeItem('authToken');
    localStorage.removeItem('currentUser');
    showAuthSection();
}

// Загрузка групп
async function loadGroups() {
    try {
        const groupsData = await apiCall('/groups');
        groups = groupsData.map(g => typeof g === 'string' ? g : g.Name);
        
        const select = document.getElementById('upload-group');
        const filterSelect = document.getElementById('filter-group');
        
        // Очищаем существующие опции (кроме первой)
        if (select) {
            while (select.children.length > 1) select.removeChild(select.lastChild);
        }
        if (filterSelect) {
            while (filterSelect.children.length > 1) filterSelect.removeChild(filterSelect.lastChild);
        }
        
        groups.forEach(group => {
            const groupName = typeof group === 'string' ? group : group.Name;
            if (select) {
                const option1 = document.createElement('option');
                option1.value = groupName;
                option1.textContent = groupName;
                select.appendChild(option1);
            }
            
            if (filterSelect) {
                const option2 = document.createElement('option');
                option2.value = groupName;
                option2.textContent = groupName;
                filterSelect.appendChild(option2);
                console.log(`Added group to filter: '${groupName}' (value: '${option2.value}')`);
            }
        });
        
        console.log(`Loaded ${groups.length} groups into filter`);
        if (filterSelect) {
            console.log('Filter select options:', Array.from(filterSelect.options).map(opt => ({value: opt.value, text: opt.text})));
        }
    } catch (error) {
        console.error('Failed to load groups:', error);
    }
}

// Загрузка банков
let banks = [];
async function loadBanks() {
    try {
        const banksData = await apiCall('/banks');
        banks = banksData.map(b => typeof b === 'string' ? b : b.Name);
        
        const select = document.getElementById('upload-bank');
        
        // Очищаем существующие опции (кроме первой)
        while (select.children.length > 1) select.removeChild(select.lastChild);
        
        banks.forEach(bank => {
            const bankName = typeof bank === 'string' ? bank : bank.Name;
            const bankValue = bankName.replace(' Bank', ''); // "Halyk Bank" -> "Halyk"
            const option = document.createElement('option');
            option.value = bankValue;
            option.textContent = bankName;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load banks:', error);
    }
}

// Загрузка банков для фильтра
async function loadBanksForFilter() {
    try {
        const banksData = await apiCall('/banks');
        const select = document.getElementById('filter-bank');
        
        // Очищаем существующие опции (кроме первой)
        while (select.children.length > 1) select.removeChild(select.lastChild);
        
        banksData.forEach(bank => {
            const bankName = typeof bank === 'string' ? bank : bank.Name;
            const bankValue = bankName.replace(' Bank', ''); // "Halyk Bank" -> "Halyk"
            const option = document.createElement('option');
            option.value = bankValue;
            option.textContent = bankName;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load banks for filter:', error);
    }
}

// Загрузка квитанций
async function loadReceipts() {
    const tbody = document.getElementById('receipts-tbody');
    tbody.innerHTML = '<tr><td colspan="9" class="loading">Загрузка...</td></tr>';

    try {
        const params = new URLSearchParams();
        
        // Универсальный поиск
        const globalSearchEl = document.getElementById('global-search');
        const globalSearchValue = globalSearchEl ? globalSearchEl.value.trim() : '';
        if (globalSearchValue) {
            params.append('search', globalSearchValue);
        }
        
        // Детальные фильтры
        const filterGroupEl = document.getElementById('filter-group');
        const filterBankEl = document.getElementById('filter-bank');
        const filterStatusEl = document.getElementById('filter-status');
        const filterDateFromEl = document.getElementById('filter-date-from');
        const filterDateToEl = document.getElementById('filter-date-to');
        
        const group = filterGroupEl ? filterGroupEl.value.trim() : '';
        const bank = filterBankEl ? filterBankEl.value.trim() : '';
        const status = filterStatusEl ? filterStatusEl.value.trim() : '';
        const dateFrom = filterDateFromEl ? filterDateFromEl.value.trim() : '';
        const dateTo = filterDateToEl ? filterDateToEl.value.trim() : '';

        console.log('Filter values - group:', group, 'bank:', bank, 'status:', status);
        console.log('Filter group element:', filterGroupEl ? filterGroupEl.value : 'not found');
        
        if (group && group !== '' && group !== 'all' && group !== 'Все группы') {
            params.append('group', group);
            console.log('Adding group filter to params:', group);
            console.log('Group filter value (encoded):', encodeURIComponent(group));
        } else {
            console.log('Group filter not applied - value:', group);
        }
        if (bank) {
            params.append('bank_type', bank);
        }
        if (status) {
            params.append('status', status);
        }
        if (dateFrom) {
            params.append('date_from', dateFrom);
        }
        if (dateTo) {
            params.append('date_to', dateTo);
        }
        
        // Пагинация
        params.append('page', currentPage.toString());
        params.append('limit', itemsPerPage.toString());

        const url = `/receipts?${params.toString()}`;
        console.log('Loading receipts with URL:', url);
        console.log('Group filter value:', group);
        console.log('All params:', params.toString());
        
        const response = await apiCall(url);
        
        console.log('Receipts response:', response);
        
        // Проверяем, является ли ответ объектом с полями receipts и total
        let receipts, total;
        if (response && typeof response.total === 'number') {
            // Обрабатываем случай, когда receipts может быть null
            receipts = response.receipts || [];
            if (!Array.isArray(receipts)) {
                receipts = [];
            }
            total = response.total;
            
            // Логируем первую квитанцию для отладки ID
            if (receipts.length > 0) {
                const firstGroup = receipts[0];
                console.log('First group:', firstGroup);
                if (firstGroup.Receipts && firstGroup.Receipts.length > 0) {
                    const firstReceipt = firstGroup.Receipts[0];
                    console.log('First receipt in group:', firstReceipt);
                    console.log('First receipt ID:', firstReceipt.ID, firstReceipt.id, firstReceipt.Id);
                    console.log('First receipt keys:', Object.keys(firstReceipt));
                } else if (firstGroup.receipts && firstGroup.receipts.length > 0) {
                    const firstReceipt = firstGroup.receipts[0];
                    console.log('First receipt in group (lowercase):', firstReceipt);
                    console.log('First receipt ID:', firstReceipt.ID, firstReceipt.id, firstReceipt.Id);
                    console.log('First receipt keys:', Object.keys(firstReceipt));
                }
            }
        } else if (Array.isArray(response)) {
            // Обратная совместимость - если ответ массив
            receipts = response;
            total = receipts.length;
            if (receipts.length > 0) {
                console.log('First receipt (non-grouped):', receipts[0]);
                console.log('First receipt ID:', receipts[0].ID, receipts[0].id, receipts[0].Id);
            }
        } else {
            console.error('Unexpected response format:', response);
            receipts = [];
            total = 0;
        }
        
        console.log(`Loaded ${receipts.length} receipts, total: ${total}`); // Для отладки
        
        // Вычисляем общее количество страниц
        totalPages = Math.max(1, Math.ceil(total / itemsPerPage));
        
        // Обновляем счетчик результатов
        const resultsCount = document.getElementById('results-count');
        if (resultsCount) {
            resultsCount.textContent = `Найдено: ${total} ${total === 1 ? 'квитанция' : total < 5 ? 'квитанции' : 'квитанций'}`;
        }
        
        // Обновляем пагинацию
        updatePagination();
        
        if (receipts.length === 0) {
            tbody.innerHTML = '<tr><td colspan="9" class="loading">Нет данных, соответствующих критериям поиска</td></tr>';
            return;
        }

        // Подсветка результатов поиска
        const searchTerm = document.getElementById('global-search').value.trim().toLowerCase();
        
        // Проверяем, является ли ответ массивом групп или массивом квитанций
        // Группа имеет поля: receipts, total_paid, payment_percent, payment_status
        // Квитанция имеет поля: ID, FullName, Group, Amount, Status
        const isGrouped = receipts.length > 0 && (
            receipts[0].Receipts !== undefined || 
            receipts[0].receipts !== undefined ||
            (receipts[0].total_paid !== undefined && receipts[0].payment_percent !== undefined)
        );
        
        console.log('Is grouped:', isGrouped, 'First item:', receipts[0]); // Для отладки
        
        if (isGrouped) {
            // Группированный вид
            tbody.innerHTML = receipts.map(group => {
                const highlightText = (text) => {
                    if (!searchTerm || !text) return text;
                    const regex = new RegExp(`(${searchTerm})`, 'gi');
                    return text.replace(regex, '<mark>$1</mark>');
                };
                
                // Определяем цвет строки
                const paymentStatus = group.PaymentStatus || group.payment_status || 'unpaid';
                const groupFullName = (group.FullName || group.full_name || '').trim();
                const hasFullName = groupFullName && groupFullName !== '-';
                
                let rowClass = '';
                let statusMessage = '';
                if (paymentStatus === 'overpaid') {
                    rowClass = 'payment-overpaid'; // Ярко зеленый для переплаты
                } else if (paymentStatus === 'paid') {
                    rowClass = 'payment-paid'; // Зеленый
                } else if (paymentStatus === 'unpaid') {
                    rowClass = 'payment-unpaid'; // Красный
                } else if (paymentStatus === 'pending') {
                    rowClass = 'payment-pending'; // Серый для необработанных
                    statusMessage = '<br><span style="color: #667eea; font-weight: 600; font-size: 0.85em;">⏳ Обработка...</span>';
                } else if (paymentStatus === 'error') {
                    // Показываем ошибку только если нет ФИО
                    if (!hasFullName) {
                        rowClass = 'payment-error'; // Красный для ошибок
                        statusMessage = '<br><span style="color: #dc3545; font-weight: 600; font-size: 0.85em;">⚠️ Ошибка распознавания</span>';
                    } else {
                        // Если есть ФИО, но статус error (сумма не распознана), не показываем ошибку
                        rowClass = 'payment-pending'; // Серый как для обработки
                    }
                } else {
                    rowClass = 'payment-partial'; // Желтый
                }
                
                const paymentPercent = (group.PaymentPercent !== undefined ? group.PaymentPercent : group.payment_percent) || 0;
                const totalPaid = (group.TotalPaid !== undefined ? group.TotalPaid : group.total_paid) || 0;
                const requiredAmount = (group.RequiredAmount !== undefined ? group.RequiredAmount : group.required_amount) || 0;
                const receiptCount = (group.Receipts || group.receipts || []).length;
                
                // Вычисляем переплату
                const overpaidAmount = requiredAmount > 0 && totalPaid > requiredAmount ? totalPaid - requiredAmount : 0;
                
                const paymentPercentStr = paymentPercent.toFixed(1);
                const totalPaidStr = totalPaid.toFixed(2);
                const requiredAmountStr = requiredAmount > 0 ? requiredAmount.toFixed(2) : '-';
                
                // Создаем уникальный ID для группы
                const fullName = (group.FullName || group.full_name || '').replace(/\s+/g, '-');
                const groupName = (group.Group || group.group || '').replace(/\s+/g, '-');
                const groupId = `group-${fullName}-${groupName}`;
                
                return `
                <tr class="${rowClass} expandable-row" data-group-id="${groupId}" onclick="toggleReceipts('${groupId}')" style="cursor: pointer;">
                    <td>
                        <button class="expand-btn" onclick="event.stopPropagation(); toggleReceipts('${groupId}')" data-expanded="false">▶</button>
                    </td>
                    <td>
                        ${highlightText((group.FullName || group.full_name) || '-')}
                        ${paymentPercent > 0 ? `<span class="payment-percent">(${paymentPercentStr}%)</span>` : ''}
                        ${statusMessage || ''}
                    </td>
                    <td>${highlightText(group.Group || group.group)}</td>
                    <td>${(group.Receipts || group.receipts) && (group.Receipts || group.receipts).length > 0 ? ((group.Receipts || group.receipts)[0].BankType || (group.Receipts || group.receipts)[0].BankType || '-') : '-'}</td>
                    <td>
                        <strong>${totalPaidStr} ₸</strong>
                        ${overpaidAmount > 0 ? `
                            <span class="overpaid-amount" style="color: #28a745; font-weight: bold; display: block; margin-top: 4px; font-size: 0.9em;">
                                Переплата: +${overpaidAmount.toFixed(2)} ₸
                            </span>
                            <button class="btn btn-success btn-sm" onclick="event.stopPropagation(); transferOverpayment('${group.FullName || group.full_name}', '${group.Group || group.group}', ${overpaidAmount.toFixed(2)})" style="margin-top: 4px; font-size: 0.8em; padding: 0.2rem 0.5rem;" title="Перенести переплату на новый учебный год">
                                🔄 Перенести
                            </button>
                        ` : ''}
                    </td>
                    <td>${requiredAmountStr} ${requiredAmountStr !== '-' ? '₸' : ''}</td>
                    <td>
                        <div class="progress-bar">
                            <div class="progress-fill" style="width: ${Math.min(paymentPercent, 100)}%; background: ${paymentStatus === 'overpaid' ? '#28a745' : paymentStatus === 'paid' ? '#28a745' : paymentStatus === 'unpaid' ? '#dc3545' : '#ffc107'};"></div>
                            <span class="progress-text">${paymentPercentStr}%</span>
                        </div>
                    </td>
                    <td>${receiptCount}</td>
                    <td onclick="event.stopPropagation();">
                        <button class="btn btn-primary btn-sm" onclick="event.stopPropagation(); viewGroupReceipts('${groupId}')" title="Просмотр всех квитанций">👁️</button>
                    </td>
                </tr>
                <tr class="receipts-detail-row" id="${groupId}-details" style="display: none;">
                    <td colspan="9">
                        <div class="receipts-details">
                            <div class="receipt-header-simple">
                                <div class="receipt-header-name">
                                    <strong>${group.FullName || group.full_name}</strong>
                                </div>
                                <div class="receipt-header-info">
                                    <span>Группа: <strong>${group.Group || group.group}</strong></span>
                                    <span>•</span>
                                    <span>Оплачено: <strong>${totalPaidStr} ₸</strong></span>
                                    <span>•</span>
                                    <span>Процент: <strong>${paymentPercentStr}%</strong></span>
                                    <span>•</span>
                                    <span>Квитанций: <strong>${receiptCount}</strong></span>
                                </div>
                            </div>
                            <table class="nested-receipts-table">
                                <thead>
                                    <tr>
                                        <th>ID</th>
                                        <th>Сумма</th>
                                        <th>Дата платежа</th>
                                        <th>Дата загрузки</th>
                                        <th>Статус</th>
                                        <th>Действия</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${(group.Receipts || group.receipts || []).map(receipt => {
                                        // Извлекаем ID - проверяем все возможные варианты
                                        let receiptId = receipt.ID;
                                        if (!receiptId) receiptId = receipt.id;
                                        if (!receiptId) receiptId = receipt.Id;
                                        if (!receiptId && receipt.Model) receiptId = receipt.Model.ID;
                                        
                                        // Логируем для отладки
                                        if (!receiptId || receiptId === 0 || receiptId === 'undefined') {
                                            console.error('Receipt ID is missing. Receipt object:', receipt);
                                            console.error('Available keys:', Object.keys(receipt));
                                            return '<tr><td colspan="6" style="color: red;">Ошибка: ID квитанции не найден</td></tr>';
                                        }
                                        const amount = receipt.Amount ? parseFloat(receipt.Amount).toFixed(2) : '0.00';
                                        const paymentDate = receipt.PaymentDate ? formatDate(receipt.PaymentDate) : '-';
                                        const uploadDate = receipt.UploadDate ? formatDate(receipt.UploadDate) : '-';
                                        const statusBadge = receipt.Status || 'pending';
                                        return `
                                        <tr onclick="viewReceipt(${receiptId})" style="cursor: pointer;" onmouseover="this.style.backgroundColor='#f0f0f0'" onmouseout="this.style.backgroundColor=''">
                                            <td>${receiptId}</td>
                                            <td><strong>${amount} ₸</strong></td>
                                            <td>${paymentDate}</td>
                                            <td>${uploadDate}</td>
                                            <td><span class="status-badge status-${statusBadge}">${getStatusText(statusBadge)}</span></td>
                                            <td onclick="event.stopPropagation();">
                                                <button class="btn btn-primary btn-sm" onclick="event.stopPropagation(); viewReceipt(${receiptId})" title="Просмотр">👁️</button>
                                                <button class="btn btn-danger btn-sm" onclick="event.stopPropagation(); deleteReceipt(${receiptId})" title="Удалить">🗑️</button>
                                            </td>
                                        </tr>
                                        `;
                                    }).join('')}
                                </tbody>
                            </table>
                        </div>
                    </td>
                </tr>
            `;
            }).join('');
        } else {
            // Старый формат (для обратной совместимости)
            tbody.innerHTML = receipts.map(receipt => {
                const paymentDate = receipt.PaymentDate ? formatDate(receipt.PaymentDate) : '-';
                const uploadDate = receipt.UploadDate ? formatDate(receipt.UploadDate) : '-';
                const amount = receipt.Amount ? parseFloat(receipt.Amount).toFixed(2) : '-';
                const status = receipt.Status || receipt.status || 'pending';
                // Извлекаем ID - проверяем все возможные варианты
                const receiptId = receipt.ID || receipt.id || receipt.Id || (receipt.Model && receipt.Model.ID) || 0;
                if (!receiptId || receiptId === 0 || receiptId === 'undefined') {
                    console.error('Receipt ID is missing in non-grouped view:', receipt);
                }
                
                // Определяем класс строки и сообщение в зависимости от статуса
                let rowClass = '';
                let statusMessage = '';
                if (status === 'error') {
                    rowClass = 'payment-error';
                    statusMessage = '<span style="color: #dc3545; font-weight: 600;">⚠️ Ошибка распознавания</span>';
                } else if (status === 'pending') {
                    rowClass = 'payment-pending';
                    statusMessage = '<span style="color: #667eea; font-weight: 600;">⏳ Обработка...</span>';
                }
                
                const highlightText = (text) => {
                    if (!searchTerm || !text) return text;
                    const regex = new RegExp(`(${searchTerm})`, 'gi');
                    return text.replace(regex, '<mark>$1</mark>');
                };
                
                return `
                <tr class="${rowClass}" onclick="${receiptId && receiptId !== 0 && receiptId !== 'undefined' ? `viewReceipt(${receiptId})` : ''}" style="cursor: ${receiptId && receiptId !== 0 && receiptId !== 'undefined' ? 'pointer' : 'default'};" onmouseover="this.style.backgroundColor='#f0f0f0'" onmouseout="this.style.backgroundColor=''">
                    <td></td>
                    <td>
                        ${highlightText(receipt.FullName || '-')}
                        ${statusMessage ? `<br>${statusMessage}` : ''}
                    </td>
                    <td>${highlightText(receipt.Group)}</td>
                    <td>${highlightText(receipt.BankType || '-')}</td>
                    <td>${amount} ${amount !== '-' ? '₸' : ''}</td>
                    <td>-</td>
                    <td>-</td>
                    <td>-</td>
                    <td onclick="event.stopPropagation();">
                        ${receiptId && receiptId !== 0 && receiptId !== 'undefined' ? `
                            <button class="btn btn-primary btn-sm" onclick="event.stopPropagation(); viewReceipt(${receiptId})" title="Просмотр квитанции">👁️</button>
                            <button class="btn btn-danger btn-sm" onclick="event.stopPropagation(); deleteReceipt(${receiptId})" title="Удалить квитанцию">🗑️</button>
                        ` : '<span style="color: red;">ID не найден</span>'}
                    </td>
                </tr>
                `;
            }).join('');
        }
    } catch (error) {
        tbody.innerHTML = '<tr><td colspan="9" class="loading">Ошибка загрузки</td></tr>';
    }
}

function getStatusText(status) {
    const statusMap = {
        'pending': 'Обработка',
        'processed': 'Обработано',
        'error': 'Ошибка'
    };
    return statusMap[status] || status;
}

function formatDate(dateString) {
    if (!dateString) return '-';
    try {
        const date = new Date(dateString);
        return date.toLocaleDateString('ru-RU');
    } catch (e) {
        return dateString;
    }
}

// Переключение отображения квитанций в группе
function toggleReceipts(groupId) {
    const detailRow = document.getElementById(`${groupId}-details`);
    if (detailRow) {
        const isHidden = detailRow.style.display === 'none' || !detailRow.style.display;
        detailRow.style.display = isHidden ? 'table-row' : 'none';
        const btn = document.querySelector(`[data-group-id="${groupId}"] .expand-btn`);
        if (btn) {
            btn.textContent = isHidden ? '▼' : '▶';
            btn.setAttribute('data-expanded', isHidden);
        }
    }
}

// Делаем функцию глобальной
window.toggleReceipts = toggleReceipts;

// Переключение раскрытия фильтров
function toggleFilters() {
    const filtersContent = document.getElementById('filters-content');
    const toggleIcon = document.querySelector('.filters-toggle-icon');
    
    if (filtersContent && toggleIcon) {
        const isHidden = filtersContent.style.display === 'none' || !filtersContent.style.display;
        filtersContent.style.display = isHidden ? 'block' : 'none';
        toggleIcon.textContent = isHidden ? '▲' : '▼';
        toggleIcon.style.transform = isHidden ? 'rotate(0deg)' : 'rotate(180deg)';
    }
}

// Делаем функцию глобальной
window.toggleFilters = toggleFilters;

function clearAllFilters() {
    const globalSearch = document.getElementById('global-search');
    const filterGroup = document.getElementById('filter-group');
    const filterBank = document.getElementById('filter-bank');
    const filterStatus = document.getElementById('filter-status');
    const filterDateFrom = document.getElementById('filter-date-from');
    const filterDateTo = document.getElementById('filter-date-to');
    const clearSearchBtn = document.getElementById('clear-search');
    
    if (globalSearch) globalSearch.value = '';
    if (filterGroup) filterGroup.value = '';
    if (filterBank) filterBank.value = '';
    if (filterStatus) filterStatus.value = '';
    if (filterDateFrom) filterDateFrom.value = '';
    if (filterDateTo) filterDateTo.value = '';
    if (clearSearchBtn) clearSearchBtn.style.display = 'none';
    
    console.log('All filters cleared');
    currentPage = 1; // Сбрасываем на первую страницу
    loadReceipts();
}

// Обновление пагинации
function updatePagination() {
    const pagination = document.getElementById('pagination');
    const prevBtn = document.getElementById('prev-page');
    const nextBtn = document.getElementById('next-page');
    const pageInfo = document.getElementById('page-info');
    
    if (totalPages <= 1) {
        pagination.style.display = 'none';
        return;
    }
    
    pagination.style.display = 'flex';
    prevBtn.disabled = currentPage === 1;
    nextBtn.disabled = currentPage >= totalPages;
    
    pageInfo.textContent = `Страница ${currentPage} из ${totalPages}`;
}

// Загрузка квитанции
async function handleUpload(e) {
    e.preventDefault();
    
    const formData = new FormData();
    const autoExtractFullname = document.getElementById('auto-extract-fullname').checked;
    const autoExtractBank = document.getElementById('auto-extract-bank').checked;
    formData.append('fullname', document.getElementById('upload-fullname').value);
    formData.append('group', document.getElementById('upload-group').value);
    formData.append('bank_type', document.getElementById('upload-bank').value);
    formData.append('auto_extract_fullname', autoExtractFullname);
    formData.append('auto_extract_bank', autoExtractBank);
    formData.append('file', document.getElementById('upload-file').files[0]);

    try {
        const response = await fetch('/api/receipts', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${authToken}`,
            },
            body: formData,
        });

        if (!response.ok) {
            let errorMessage = 'Ошибка загрузки';
            try {
                const contentType = response.headers.get('content-type');
                if (contentType && contentType.includes('application/json')) {
                    const text = await response.text();
                    if (text && text.trim()) {
                        const error = JSON.parse(text);
                        errorMessage = error.error || errorMessage;
                    }
                } else {
                    errorMessage = `Ошибка ${response.status}: ${response.statusText}`;
                }
            } catch (e) {
                errorMessage = `Ошибка ${response.status}: ${response.statusText}`;
            }
            throw new Error(errorMessage);
        }
        
        // Проверяем, есть ли JSON в успешном ответе
        let receiptData = null;
        try {
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                const text = await response.text();
                if (text && text.trim()) {
                    receiptData = JSON.parse(text);
                }
            }
        } catch (e) {
            console.warn('Could not parse response as JSON:', e);
        }

        // Показываем уведомление об успешной загрузке
        showNotification('✅', 'Квитанция успешно загружена! Обработка началась. Данные будут обновлены автоматически после завершения обработки.', 'success');
        
        const modal = document.getElementById('upload-modal');
        closeModal('upload-modal');
        const uploadForm = document.getElementById('upload-form');
        if (uploadForm) {
            uploadForm.reset();
            // После сброса формы снова включаем чекбоксы
            const autoExtractFullname = document.getElementById('auto-extract-fullname');
            const autoExtractBank = document.getElementById('auto-extract-bank');
            if (autoExtractFullname) {
                autoExtractFullname.checked = true;
                autoExtractFullname.dispatchEvent(new Event('change'));
            }
            if (autoExtractBank) {
                autoExtractBank.checked = true;
                autoExtractBank.dispatchEvent(new Event('change'));
            }
        }
        const fileNameEl = document.getElementById('file-name');
        if (fileNameEl) {
            fileNameEl.style.display = 'none';
        }
        const fileUploadArea = document.getElementById('file-upload-area');
        if (fileUploadArea) {
            fileUploadArea.style.borderColor = '#667eea';
            fileUploadArea.style.background = '#f0f4ff';
        }
        loadReceipts();
        
        // Автоматически обновляем список каждые 5 секунд в течение 30 секунд
        let updateCount = 0;
        const updateInterval = setInterval(() => {
            updateCount++;
            loadReceipts();
            if (updateCount >= 6) { // 6 раз по 5 секунд = 30 секунд
                clearInterval(updateInterval);
            }
        }, 5000);
    } catch (error) {
        showNotification('❌', error.message || 'Ошибка при загрузке квитанции', 'error');
    }
}

// Функция для показа уведомлений
function showNotification(icon, message, type = 'success') {
    const modal = document.getElementById('notification-modal');
    const iconEl = document.getElementById('notification-icon');
    const messageEl = document.getElementById('notification-message');
    
    if (!modal || !iconEl || !messageEl) {
        // Fallback на alert, если модальное окно не найдено
        showNotification('⚠️', message, 'error');
        return;
    }
    
    iconEl.textContent = icon;
    messageEl.textContent = message;
    
    // Устанавливаем цвет иконки в зависимости от типа
    if (type === 'success') {
        iconEl.style.color = '#28a745';
    } else if (type === 'error') {
        iconEl.style.color = '#dc3545';
    } else if (type === 'info') {
        iconEl.style.color = '#17a2b8';
    } else {
        iconEl.style.color = '#667eea';
    }
    
    modal.style.display = 'flex';
    setTimeout(() => modal.classList.add('show'), 10);
    
    // Автоматически закрываем через 4 секунды
    setTimeout(() => {
        if (modal && modal.style.display === 'flex') {
            modal.classList.remove('show');
            setTimeout(() => {
                if (modal) modal.style.display = 'none';
            }, 300);
        }
    }, 4000);
}

// Функция для показа диалога подтверждения (глобальная)
function showConfirmDialog(title, message, confirmText = 'Да', cancelText = 'Нет') {
    return new Promise((resolve) => {
        const modal = document.createElement('div');
        modal.className = 'modal confirm-modal';
        modal.style.display = 'flex';
        modal.style.zIndex = '10001';
        modal.innerHTML = `
            <div class="modal-content confirm-content">
                <div class="modal-header">
                    <h2>${title}</h2>
                    <button class="close" type="button">&times;</button>
                </div>
                <div class="modal-body">
                    <p style="font-size: 1.1rem; color: #333; margin: 1rem 0; line-height: 1.6;">${message}</p>
                </div>
                <div class="modal-footer" style="display: flex; gap: 1rem; justify-content: flex-end; padding: 1rem 1.5rem; border-top: 1px solid #e0e0e0; margin-top: 1rem;">
                    <button class="btn btn-secondary confirm-cancel">${cancelText}</button>
                    <button class="btn btn-danger confirm-ok">${confirmText}</button>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        const closeModal = (result) => {
            modal.classList.remove('show');
            setTimeout(() => {
                modal.style.display = 'none';
                if (document.body.contains(modal)) {
                    document.body.removeChild(modal);
                }
            }, 300);
            resolve(result);
        };
        
        const okBtn = modal.querySelector('.confirm-ok');
        const cancelBtn = modal.querySelector('.confirm-cancel');
        const closeBtn = modal.querySelector('.close');
        
        if (okBtn) {
            okBtn.addEventListener('click', () => closeModal(true));
        }
        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => closeModal(false));
        }
        if (closeBtn) {
            closeBtn.addEventListener('click', () => closeModal(false));
        }
        
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                closeModal(false);
            }
        });
        
        setTimeout(() => {
            if (document.body.contains(modal)) {
                modal.classList.add('show');
            }
        }, 10);
    });
}

// Делаем функцию глобальной для доступа из onclick
window.showConfirmDialog = showConfirmDialog;

// Обработчик закрытия уведомления
document.addEventListener('DOMContentLoaded', () => {
    const notificationClose = document.getElementById('notification-close');
    if (notificationClose) {
        notificationClose.addEventListener('click', () => {
            const modal = document.getElementById('notification-modal');
            modal.style.display = 'none';
            modal.classList.remove('show');
        });
    }
});

// Просмотр всех квитанций группы (использует ту же функцию toggleReceipts)
function viewGroupReceipts(groupId) {
    toggleReceipts(groupId);
}

// Делаем функцию глобальной
window.viewGroupReceipts = viewGroupReceipts;

// Просмотр квитанции
async function viewReceipt(id) {
    if (!id || id === 0 || id === 'undefined' || id === undefined) {
        showNotification('❌', 'Ошибка: ID квитанции не указан', 'error');
        console.error('viewReceipt called with invalid ID:', id);
        return;
    }
    
    try {
        const receipt = await apiCall(`/receipts/${id}`);
        
        const details = document.getElementById('receipt-details');
        const amount = receipt.Amount ? parseFloat(receipt.Amount).toFixed(2) : '-';
        const receiptId = receipt.ID || receipt.id || receipt.Id;
        details.innerHTML = `
            <div class="info-grid">
                <div class="info-item">
                    <strong>ID</strong>
                    <span>${receiptId}</span>
                </div>
                <div class="info-item">
                    <strong>ФИО</strong>
                    <span>${receipt.FullName || '-'}</span>
                </div>
                <div class="info-item">
                    <strong>Группа</strong>
                    <span>${receipt.Group}</span>
                </div>
                <div class="info-item">
                    <strong>Тип банка</strong>
                    <span>${receipt.BankType}</span>
                </div>
                <div class="info-item">
                    <strong>Сумма</strong>
                    <span>${amount} ${amount !== '-' ? '₸' : ''}</span>
                </div>
                <div class="info-item">
                    <strong>Дата платежа</strong>
                    <span>${receipt.PaymentDate ? formatDate(receipt.PaymentDate) : '-'}</span>
                </div>
                <div class="info-item">
                    <strong>Дата загрузки</strong>
                    <span>${receipt.UploadDate ? formatDate(receipt.UploadDate) : '-'}</span>
                </div>
                <div class="info-item">
                    <strong>Статус</strong>
                    <span><span class="status-badge status-${receipt.Status}">${getStatusText(receipt.Status)}</span></span>
                </div>
                <div class="info-item" style="grid-column: 1 / -1;">
                    <strong>Наименование</strong>
                    <span>${receipt.UniversityName || 'Не указано'}</span>
                </div>
                <div class="info-item" style="grid-column: 1 / -1;">
                    <strong>📁 Путь к файлу</strong>
                    <code style="word-break: break-all; font-size: 0.85rem;">${receipt.FilePath || '-'}</code>
                </div>
            </div>
        `;

        const imageContainer = document.getElementById('receipt-image-container');
        // Загружаем файл с авторизацией через fetch
        imageContainer.innerHTML = '<div style="text-align: center; padding: 2rem;"><p>Загрузка файла...</p></div>';
        
        try {
            const fileUrl = `/api/receipts/${id}/file`;
            const response = await fetch(fileUrl, {
                headers: {
                    'Authorization': `Bearer ${authToken}`
                }
            });
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const blob = await response.blob();
            const fileObjectUrl = URL.createObjectURL(blob);
            
            // Определяем тип файла по расширению или MIME типу
            const filePath = receipt.FilePath || '';
            const isPdf = filePath.toLowerCase().endsWith('.pdf') || blob.type === 'application/pdf';
            
            if (isPdf) {
                // Для PDF используем iframe (более надежно, чем embed)
                const fileName = filePath.split('/').pop() || 'receipt.pdf';
                imageContainer.innerHTML = `
                    <div style="text-align: center; padding: 1rem;">
                        <iframe src="${fileObjectUrl}" 
                                type="application/pdf"
                                style="width: 100%; height: 600px; border: 1px solid #ddd; border-radius: 8px; box-shadow: 0 5px 20px rgba(0, 0, 0, 0.15);"
                                title="Квитанция PDF"
                                onload="setTimeout(() => { try { URL.revokeObjectURL('${fileObjectUrl}'); } catch(e) {} }, 1000);">
                            <p>Ваш браузер не поддерживает отображение PDF. 
                               <a href="${fileObjectUrl}" target="_blank" download="${fileName}">Скачайте файл</a> для просмотра.</p>
                        </iframe>
                        <div style="margin-top: 1rem;">
                            <a href="${fileObjectUrl}" 
                               target="_blank" 
                               download="${fileName}"
                               class="btn btn-primary"
                               style="text-decoration: none; display: inline-block;">
                                📥 Скачать PDF
                            </a>
                        </div>
                    </div>
                `;
            } else {
                // Для изображений используем img
                imageContainer.innerHTML = `
                    <div style="text-align: center; padding: 1rem;">
                        <img src="${fileObjectUrl}" alt="Квитанция" 
                             onerror="this.onerror=null; this.parentElement.innerHTML='<p style=\\'color: #dc3545; padding: 2rem;\\'>⚠️ Не удалось загрузить изображение квитанции</p><p style=\\'color: #666; font-size: 0.9rem;\\'>Файл: ${receipt.FilePath || 'не указан'}</p><p style=\\'color: #666; font-size: 0.9rem;\\'>Тип: ${blob.type || 'неизвестно'}</p>';"
                             style="max-width: 100%; max-height: 600px; border-radius: 8px; box-shadow: 0 5px 20px rgba(0, 0, 0, 0.15); cursor: pointer;"
                             onclick="window.open('${fileObjectUrl}', '_blank')"
                             title="Нажмите для открытия в полном размере"
                             onload="URL.revokeObjectURL('${fileObjectUrl}')">
                    </div>
                `;
            }
        } catch (error) {
            console.error('Error loading receipt file:', error);
            imageContainer.innerHTML = `
                <div style="text-align: center; padding: 2rem;">
                    <p style="color: #dc3545; margin-bottom: 0.5rem;">⚠️ Не удалось загрузить файл квитанции</p>
                    <p style="color: #666; font-size: 0.9rem; margin-bottom: 0.5rem;">Ошибка: ${error.message}</p>
                    <p style="color: #666; font-size: 0.9rem;">Файл: ${receipt.FilePath || 'не указан'}</p>
                </div>
            `;
        }

        const modal = document.getElementById('view-modal');
        modal.style.display = 'flex';
        modal.classList.add('show');
    } catch (error) {
        showNotification('❌', 'Ошибка загрузки квитанции', 'error');
    }
}

// Удаление квитанции (глобальная функция для доступа из onclick)
async function deleteReceipt(id) {
    try {
        // Показываем красивое подтверждение
        const confirmed = await showConfirmDialog(
            '🗑️ Удаление квитанции',
            'Вы уверены, что хотите удалить эту квитанцию? Файл также будет удален. Это действие нельзя отменить.',
            'Удалить',
            'Отмена'
        );
        
        if (!confirmed) {
            return;
        }

        await apiCall(`/receipts/${id}`, {
            method: 'DELETE',
        });

        showNotification('✅', 'Квитанция успешно удалена!', 'success');
        loadReceipts();
    } catch (error) {
        console.error('Ошибка при удалении квитанции:', error);
        showNotification('❌', 'Ошибка удаления квитанции: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

// Делаем функцию глобальной для доступа из onclick
window.deleteReceipt = deleteReceipt;

// Экспорт в Excel
async function handleExport() {
    try {
        showNotification('⏳', 'Экспорт данных...', 'info');
        
        const params = new URLSearchParams();
        
        // Универсальный поиск
        const globalSearchEl = document.getElementById('global-search');
        const globalSearchValue = globalSearchEl ? globalSearchEl.value.trim() : '';
        if (globalSearchValue) {
            params.append('search', globalSearchValue);
        }
        
        // Детальные фильтры
        const filterGroupEl = document.getElementById('filter-group');
        const filterBankEl = document.getElementById('filter-bank');
        const filterStatusEl = document.getElementById('filter-status');
        const filterDateFromEl = document.getElementById('filter-date-from');
        const filterDateToEl = document.getElementById('filter-date-to');
        
        const group = filterGroupEl ? filterGroupEl.value.trim() : '';
        const bank = filterBankEl ? filterBankEl.value.trim() : '';
        const status = filterStatusEl ? filterStatusEl.value.trim() : '';
        const dateFrom = filterDateFromEl ? filterDateFromEl.value.trim() : '';
        const dateTo = filterDateToEl ? filterDateToEl.value.trim() : '';

        if (group) params.append('group', group);
        if (bank) params.append('bank_type', bank);
        if (status) params.append('status', status);
        if (dateFrom) params.append('date_from', dateFrom);
        if (dateTo) params.append('date_to', dateTo);

        const response = await fetch(`/api/admin/export?${params.toString()}`, {
            headers: {
                'Authorization': `Bearer ${authToken}`,
            },
        });

        if (!response.ok) {
            let errorMessage = 'Ошибка экспорта';
            try {
                const contentType = response.headers.get('content-type');
                if (contentType && contentType.includes('application/json')) {
                    const text = await response.text();
                    if (text && text.trim()) {
                        const error = JSON.parse(text);
                        errorMessage = error.error || errorMessage;
                    }
                } else {
                    errorMessage = `Ошибка ${response.status}: ${response.statusText}`;
                }
            } catch (e) {
                errorMessage = `Ошибка ${response.status}: ${response.statusText}`;
            }
            throw new Error(errorMessage);
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'receipts.xlsx';
        a.click();
        window.URL.revokeObjectURL(url);
        
        showNotification('✅', 'Экспорт успешно завершен!', 'success');
    } catch (error) {
        console.error('Export error:', error);
        showNotification('❌', 'Ошибка экспорта: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

// Управление пользователями
async function loadUsers() {
    const tbody = document.getElementById('users-tbody');
    tbody.innerHTML = '<tr><td colspan="4" class="loading">Загрузка...</td></tr>';

    try {
        const users = await apiCall('/admin/users');
        
        if (users.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" class="loading">Нет пользователей</td></tr>';
            return;
        }

        tbody.innerHTML = users.map(user => {
            const userId = user.ID || user.id || user.Id;
            return `
            <tr>
                <td>${userId}</td>
                <td>${user.Username}</td>
                <td>${user.Role === 'admin' ? 'Администратор' : 'Пользователь'}</td>
                <td>
                    <button class="btn btn-danger btn-sm" onclick="deleteUser(${userId})">Удалить</button>
                </td>
            </tr>
        `;
        }).join('');
    } catch (error) {
        tbody.innerHTML = '<tr><td colspan="4" class="loading">Ошибка загрузки</td></tr>';
    }
}

async function handleAddUser(e) {
    e.preventDefault();
    
    const username = document.getElementById('add-user-username').value;
    const password = document.getElementById('add-user-password').value;
    const role = document.getElementById('add-user-role').value;

    try {
        await apiCall('/admin/users', {
            method: 'POST',
            body: JSON.stringify({ username, password, role }),
        });

        showNotification('✅', 'Пользователь успешно создан!', 'success');
        closeModal('add-user-modal');
        document.getElementById('add-user-form').reset();
        loadUsers();
    } catch (error) {
        // Ошибка уже обработана
    }
}

async function deleteUser(id) {
    const confirmed = await showConfirmDialog(
        'Удаление пользователя',
        'Вы уверены, что хотите удалить этого пользователя?',
        'Удалить',
        'Отмена'
    );
    
    if (!confirmed) {
        return;
    }

    try {
        await apiCall(`/admin/users/${id}`, {
            method: 'DELETE',
        });

        showNotification('✅', 'Пользователь успешно удален!', 'success');
        loadUsers();
    } catch (error) {
        // Ошибка уже обработана
    }
}

// Управление группами
async function loadGroupsForAdmin() {
    const tbody = document.getElementById('groups-tbody');
    tbody.innerHTML = '<tr><td colspan="2" class="loading">Загрузка...</td></tr>';

    try {
        const groupsData = await apiCall('/groups');
        
        if (groupsData.length === 0) {
            tbody.innerHTML = '<tr><td colspan="2" class="loading">Нет групп</td></tr>';
            return;
        }

        tbody.innerHTML = groupsData.map(group => {
            const groupId = group.ID || group.id || group.Id;
            const groupName = group.Name || group;
            return `
            <tr>
                <td>${groupName}</td>
                <td>
                    <button class="btn btn-primary btn-sm" onclick="editGroup(${groupId}, '${groupName.replace(/'/g, "\\'")}')">Редактировать</button>
                    <button class="btn btn-danger btn-sm" onclick="deleteGroup(${groupId})">Удалить</button>
                </td>
            </tr>
        `;
        }).join('');
    } catch (error) {
        tbody.innerHTML = '<tr><td colspan="2" class="loading">Ошибка загрузки</td></tr>';
    }
}

async function handleSaveGroup(e) {
    e.preventDefault();
    
    const id = document.getElementById('edit-group-id').value;
    const name = document.getElementById('edit-group-name').value;

    try {
        if (id) {
            await apiCall(`/admin/groups/${id}`, {
                method: 'PUT',
                body: JSON.stringify({ name }),
            });
            showNotification('✅', 'Группа успешно обновлена!', 'success');
        } else {
            await apiCall('/admin/groups', {
                method: 'POST',
                body: JSON.stringify({ name }),
            });
            showNotification('✅', 'Группа успешно создана!', 'success');
        }
        
        closeModal('edit-group-modal');
        document.getElementById('edit-group-form').reset();
        loadGroupsForAdmin();
        loadGroups();
    } catch (error) {
        // Ошибка уже обработана
    }
}

// Делаем функцию глобальной для доступа из onclick
window.deleteUser = deleteUser;

function editGroup(id, name) {
    document.getElementById('edit-group-id').value = id;
    document.getElementById('edit-group-name').value = name;
    document.getElementById('edit-group-title').textContent = '📚 Редактировать группу';
    const modal = document.getElementById('edit-group-modal');
    modal.style.display = 'flex';
    modal.classList.add('show');
}

async function deleteGroup(id) {
    const confirmed = await showConfirmDialog(
        'Удаление группы',
        'Вы уверены, что хотите удалить эту группу?',
        'Удалить',
        'Отмена'
    );
    
    if (!confirmed) {
        return;
    }

    try {
        await apiCall(`/admin/groups/${id}`, {
            method: 'DELETE',
        });

        showNotification('✅', 'Группа успешно удалена!', 'success');
        loadGroupsForAdmin();
        loadGroups();
    } catch (error) {
        // Ошибка уже обработана
    }
}

// Управление банками
async function loadBanksForAdmin() {
    const tbody = document.getElementById('banks-tbody');
    tbody.innerHTML = '<tr><td colspan="2" class="loading">Загрузка...</td></tr>';

    try {
        const banksData = await apiCall('/banks');
        
        if (banksData.length === 0) {
            tbody.innerHTML = '<tr><td colspan="2" class="loading">Нет банков</td></tr>';
            return;
        }

        tbody.innerHTML = banksData.map(bank => {
            const bankId = bank.ID || bank.id || bank.Id;
            const bankName = bank.Name || bank;
            return `
            <tr>
                <td>${bankName}</td>
                <td>
                    <button class="btn btn-primary btn-sm" onclick="editBank(${bankId}, '${bankName.replace(/'/g, "\\'")}')">Редактировать</button>
                    <button class="btn btn-danger btn-sm" onclick="deleteBank(${bankId})">Удалить</button>
                </td>
            </tr>
        `;
        }).join('');
    } catch (error) {
        tbody.innerHTML = '<tr><td colspan="2" class="loading">Ошибка загрузки</td></tr>';
    }
}

async function handleSaveBank(e) {
    e.preventDefault();
    
    const id = document.getElementById('edit-bank-id').value;
    const name = document.getElementById('edit-bank-name').value;

    try {
        if (id) {
            await apiCall(`/admin/banks/${id}`, {
                method: 'PUT',
                body: JSON.stringify({ name }),
            });
            showNotification('✅', 'Банк успешно обновлен!', 'success');
        } else {
            await apiCall('/admin/banks', {
                method: 'POST',
                body: JSON.stringify({ name }),
            });
            showNotification('✅', 'Банк успешно создан!', 'success');
        }
        
        closeModal('edit-bank-modal');
        document.getElementById('edit-bank-form').reset();
        loadBanksForAdmin();
        loadBanks();
    } catch (error) {
        // Ошибка уже обработана
    }
}

// Делаем функцию глобальной для доступа из onclick
window.deleteGroup = deleteGroup;

function editBank(id, name) {
    document.getElementById('edit-bank-id').value = id;
    document.getElementById('edit-bank-name').value = name;
    document.getElementById('edit-bank-title').textContent = '🏦 Редактировать банк';
    const modal = document.getElementById('edit-bank-modal');
    modal.style.display = 'flex';
    modal.classList.add('show');
}

async function deleteBank(id) {
    const confirmed = await showConfirmDialog(
        'Удаление банка',
        'Вы уверены, что хотите удалить этот банк?',
        'Удалить',
        'Отмена'
    );
    
    if (!confirmed) {
        return;
    }

    try {
        await apiCall(`/admin/banks/${id}`, {
            method: 'DELETE',
        });

        showNotification('✅', 'Банк успешно удален!', 'success');
        loadBanksForAdmin();
        loadBanks();
    } catch (error) {
        // Ошибка уже обработана
    }
}

// Делаем функцию глобальной для доступа из onclick
window.deleteBank = deleteBank;

// Управление учебными годами
async function loadAcademicYearsForAdmin() {
    const tbody = document.getElementById('academic-years-tbody');
    tbody.innerHTML = '<tr><td colspan="5" class="loading">Загрузка...</td></tr>';

    try {
        const yearsData = await apiCall('/academic-years');
        
        if (yearsData.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="loading">Нет учебных годов</td></tr>';
            return;
        }

        tbody.innerHTML = yearsData.map(year => {
            const yearId = year.ID || year.id || year.Id;
            const startDate = year.StartDate ? formatDate(year.StartDate) : '-';
            const endDate = year.EndDate ? formatDate(year.EndDate) : '-';
            // Экранируем значения для безопасной передачи в onclick
            const yearEscaped = (year.Year || '').replace(/'/g, "\\'");
            const startDateEscaped = (year.StartDate || '').replace(/'/g, "\\'");
            const endDateEscaped = (year.EndDate || '').replace(/'/g, "\\'");
            
            return `
            <tr>
                <td><strong>${year.Year}</strong></td>
                <td>${startDate}</td>
                <td>${endDate}</td>
                <td>${year.IsActive ? '<span style="color: #28a745; font-weight: 600;">✓ Активен</span>' : '<span style="color: #999;">Неактивен</span>'}</td>
                <td>
                    <button class="btn btn-primary btn-sm" onclick="editAcademicYear(${yearId}, '${yearEscaped}', '${startDateEscaped}', '${endDateEscaped}', ${year.IsActive})">Редактировать</button>
                    <button class="btn btn-danger btn-sm" onclick="deleteAcademicYear(${yearId})">Удалить</button>
                </td>
            </tr>
        `;
        }).join('');
    } catch (error) {
        tbody.innerHTML = '<tr><td colspan="5" class="loading">Ошибка загрузки</td></tr>';
    }
}

async function loadGroupsForAmountForm() {
    try {
        const groupsData = await apiCall('/groups');
        const select = document.getElementById('set-amount-group');
        
        if (!select) return;
        
        // Очищаем все опции кроме первой (если есть placeholder)
        while (select.firstChild) select.removeChild(select.firstChild);
        
        // Добавляем placeholder
        const placeholder = document.createElement('option');
        placeholder.value = '';
        placeholder.textContent = 'Выберите группу';
        placeholder.disabled = true;
        placeholder.selected = true;
        select.appendChild(placeholder);
        
        groupsData.forEach(group => {
            const groupName = typeof group === 'string' ? group : group.Name;
            const groupId = typeof group === 'string' ? null : (group.ID || group.id || group.Id);
            if (groupId) {
                const option = document.createElement('option');
                option.value = groupId;
                option.textContent = groupName;
                select.appendChild(option);
            }
        });
    } catch (error) {
        console.error('Failed to load groups for amount form:', error);
    }
}

async function loadAcademicYearsForAmountForm() {
    try {
        const yearsData = await apiCall('/academic-years');
        const select = document.getElementById('set-amount-year');
        
        if (!select) return;
        
        // Очищаем все опции кроме первой (если есть placeholder)
        while (select.firstChild) select.removeChild(select.firstChild);
        
        // Добавляем placeholder
        const placeholder = document.createElement('option');
        placeholder.value = '';
        placeholder.textContent = 'Выберите учебный год';
        placeholder.disabled = true;
        placeholder.selected = true;
        select.appendChild(placeholder);
        
        yearsData.forEach(year => {
            const yearId = year.ID || year.id || year.Id;
            const option = document.createElement('option');
            option.value = yearId;
            option.textContent = year.Year;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load academic years for amount form:', error);
    }
}

async function handleSaveAcademicYear(e) {
    e.preventDefault();
    
    const id = document.getElementById('edit-academic-year-id').value;
    const year = document.getElementById('edit-academic-year-year').value;
    const startDate = document.getElementById('edit-academic-year-start').value;
    const endDate = document.getElementById('edit-academic-year-end').value;
    const isActive = document.getElementById('edit-academic-year-active').checked;

    try {
        const data = {
            year: year,
            start_date: startDate,
            end_date: endDate,
            is_active: isActive
        };
        
        if (id) {
            await apiCall(`/admin/academic-years/${id}`, {
                method: 'PUT',
                body: JSON.stringify(data),
            });
            showNotification('✅', 'Учебный год успешно обновлен!', 'success');
        } else {
            await apiCall('/admin/academic-years', {
                method: 'POST',
                body: JSON.stringify(data),
            });
            showNotification('✅', 'Учебный год успешно создан!', 'success');
        }
        
        closeModal('edit-academic-year-modal');
        document.getElementById('edit-academic-year-form').reset();
        loadAcademicYearsForAdmin();
        loadAcademicYearsForAmountForm();
    } catch (error) {
        // Ошибка уже обработана
    }
}

function editAcademicYear(id, year, startDate, endDate, isActive) {
    document.getElementById('edit-academic-year-id').value = id;
    document.getElementById('edit-academic-year-year').value = year || '';
    
    // Обработка даты начала
    if (startDate) {
        const startDateStr = typeof startDate === 'string' ? startDate : startDate.toISOString();
        document.getElementById('edit-academic-year-start').value = startDateStr.split('T')[0];
    } else {
        document.getElementById('edit-academic-year-start').value = '';
    }
    
    // Обработка даты окончания
    if (endDate) {
        const endDateStr = typeof endDate === 'string' ? endDate : endDate.toISOString();
        document.getElementById('edit-academic-year-end').value = endDateStr.split('T')[0];
    } else {
        document.getElementById('edit-academic-year-end').value = '';
    }
    
    document.getElementById('edit-academic-year-active').checked = isActive || false;
    document.getElementById('edit-academic-year-title').textContent = '📅 Редактировать учебный год';
    const modal = document.getElementById('edit-academic-year-modal');
    modal.style.display = 'flex';
    modal.classList.add('show');
}

async function deleteAcademicYear(id) {
    const confirmed = await showConfirmDialog(
        'Удаление учебного года',
        'Вы уверены, что хотите удалить этот учебный год?',
        'Удалить',
        'Отмена'
    );
    
    if (!confirmed) {
        return;
    }

    try {
        await apiCall(`/admin/academic-years/${id}`, {
            method: 'DELETE',
        });

        showNotification('✅', 'Учебный год успешно удален!', 'success');
        loadAcademicYearsForAdmin();
        loadAcademicYearsForAmountForm();
    } catch (error) {
        // Ошибка уже обработана
    }
}

// Делаем функцию глобальной для доступа из onclick
window.deleteAcademicYear = deleteAcademicYear;

async function handleSetGroupAmount(e) {
    e.preventDefault();
    
    const editId = document.getElementById('edit-group-amount-id').value;
    const groupId = document.getElementById('set-amount-group').value;
    const yearId = document.getElementById('set-amount-year').value;
    const amount = parseFloat(document.getElementById('set-amount-value').value);

    try {
        await apiCall('/admin/academic-years/group-amount', {
            method: 'POST',
            body: JSON.stringify({
                group_id: parseInt(groupId),
                academic_year_id: parseInt(yearId),
                required_amount: amount
            }),
        });
        
        showNotification('✅', 'Требуемая сумма успешно сохранена!', 'success');
        document.getElementById('set-group-amount-form').reset();
        document.getElementById('edit-group-amount-id').value = '';
        document.getElementById('group-amount-form-container').style.display = 'none';
        loadGroupRequiredAmounts();
    } catch (error) {
        // Ошибка уже обработана
    }
}

// Загрузка всех установленных требуемых сумм
async function loadGroupRequiredAmounts() {
    const tbody = document.getElementById('group-amounts-tbody');
    if (!tbody) return;
    
    tbody.innerHTML = '<tr><td colspan="5" class="loading">Загрузка...</td></tr>';

    try {
        const amounts = await apiCall('/admin/academic-years/group-amounts');
        
        if (!amounts || amounts.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="loading">Нет установленных сумм</td></tr>';
            return;
        }

        tbody.innerHTML = amounts.map(item => {
            const amountId = item.id || item.ID;
            const groupName = item.group_name || item.GroupName || '-';
            const academicYear = item.academic_year || item.AcademicYear || '-';
            const requiredAmount = (item.required_amount !== undefined ? item.required_amount : item.RequiredAmount) || 0;
            const updatedAt = item.updated_at || item.UpdatedAt || '-';
            
            return `
                <tr>
                    <td><strong>${groupName}</strong></td>
                    <td>${academicYear}</td>
                    <td><strong style="color: #667eea;">${requiredAmount.toFixed(2)} ₸</strong></td>
                    <td>${updatedAt}</td>
                    <td>
                        <button class="btn btn-primary btn-sm" onclick="editGroupAmount(${amountId}, '${groupName.replace(/'/g, "\\'")}', ${item.academic_year_id || item.AcademicYearID}, ${item.group_id || item.GroupID}, ${requiredAmount})" title="Редактировать">✏️</button>
                        <button class="btn btn-danger btn-sm" onclick="deleteGroupAmount(${amountId})" title="Удалить">🗑️</button>
                    </td>
                </tr>
            `;
        }).join('');
    } catch (error) {
        console.error('Error loading group amounts:', error);
        tbody.innerHTML = '<tr><td colspan="5" class="loading">Ошибка загрузки</td></tr>';
    }
}

// Редактирование требуемой суммы
function editGroupAmount(id, groupName, academicYearId, groupId, amount) {
    document.getElementById('edit-group-amount-id').value = id;
    document.getElementById('set-amount-value').value = amount;
    document.getElementById('set-amount-year').value = academicYearId;
    document.getElementById('set-amount-group').value = groupId;
    document.getElementById('group-amount-form-title').textContent = `✏️ Редактировать требуемую сумму для группы "${groupName}"`;
    document.getElementById('group-amount-form-container').style.display = 'block';
    
    // Прокручиваем к форме
    document.getElementById('group-amount-form-container').scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

// Удаление требуемой суммы
async function deleteGroupAmount(id) {
    const confirmed = await showConfirmDialog(
        'Удаление требуемой суммы',
        'Вы уверены, что хотите удалить эту требуемую сумму?',
        'Удалить',
        'Отмена'
    );
    
    if (!confirmed) {
        return;
    }

    try {
        await apiCall(`/admin/academic-years/group-amount/${id}`, {
            method: 'DELETE',
        });

        showNotification('✅', 'Требуемая сумма успешно удалена!', 'success');
        loadGroupRequiredAmounts();
    } catch (error) {
        // Ошибка уже обработана
    }
}

// Делаем функции глобальными
window.editGroupAmount = editGroupAmount;
window.deleteGroupAmount = deleteGroupAmount;

// ========== ОТЧЕТЫ ==========

let currentReportData = null;

// Загрузка учебных годов для отчета
async function loadReportYears() {
    try {
        const years = await apiCall('/academic-years');
        const select = document.getElementById('report-year-select');
        if (!select) return;
        
        select.innerHTML = '<option value="">Выберите учебный год</option>';
        years.forEach(year => {
            const option = document.createElement('option');
            option.value = year.ID || year.id || year.Id;
            option.textContent = year.Year;
            if (year.IsActive) {
                option.selected = true;
            }
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading report years:', error);
    }
}

// Загрузка групп для отчета
async function loadReportGroups() {
    try {
        const groups = await apiCall('/groups');
        const select = document.getElementById('report-group-select');
        if (!select) return;
        
        select.innerHTML = '<option value="">Выберите группу</option>';
        groups.forEach(group => {
            const option = document.createElement('option');
            option.value = group.Name;
            option.textContent = group.Name;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading report groups:', error);
    }
}

// Загрузка отчета по группе
async function loadGroupReport() {
    const yearSelect = document.getElementById('report-year-select');
    const groupSelect = document.getElementById('report-group-select');
    const reportContent = document.getElementById('report-content');
    const exportBtn = document.getElementById('export-report-btn');
    
    if (!yearSelect || !groupSelect || !reportContent) return;
    
    const yearId = yearSelect.value;
    const groupName = groupSelect.value;
    
    if (!yearId || !groupName) {
        showNotification('⚠️', 'Выберите учебный год и группу', 'error');
        return;
    }
    
    try {
        showNotification('⏳', 'Загрузка отчета...', 'info');
        
        const report = await apiCall(`/admin/reports/group?group=${encodeURIComponent(groupName)}&academic_year_id=${yearId}`);
        
        currentReportData = report;
        
        // Обновляем заголовок
        const groupNameEl = document.getElementById('report-group-name');
        const requiredAmountEl = document.getElementById('report-required-amount');
        if (groupNameEl) groupNameEl.textContent = `Группа: ${groupName}`;
        if (requiredAmountEl) requiredAmountEl.textContent = report.required_amount || 0;
        
        // Заполняем таблицу
        const tbody = document.getElementById('report-tbody');
        if (!tbody) return;
        
        tbody.innerHTML = '';
        
        const students = report.students || [];
        students.forEach(student => {
            const statusMap = {
                'paid': 'Оплачено',
                'partial': 'Частично',
                'unpaid': 'Не оплачено',
                'overpaid': 'Переплата',
                'pending': 'Обработка'
            };
            
            const status = statusMap[student.payment_status] || student.payment_status;
            const statusClass = student.payment_status === 'paid' ? 'status-processed' : 
                              student.payment_status === 'overpaid' ? 'status-processed' :
                              student.payment_status === 'partial' ? 'status-pending' : 'status-error';
            
            const appliedDate = student.applied_date ? formatDate(student.applied_date) : '-';
            
            const row = document.createElement('tr');
            row.innerHTML = `
                <td><strong>${student.full_name}</strong></td>
                <td>${student.total_paid.toFixed(2)} ₸</td>
                <td>
                    ${student.applied_amount > 0 ? `
                        <span style="color: #28a745; font-weight: bold;">${student.applied_amount.toFixed(2)} ₸</span>
                        ${student.applied_by ? `<br><small style="color: #666;">Применил: ${student.applied_by}</small>` : ''}
                        ${appliedDate !== '-' ? `<br><small style="color: #666;">${appliedDate}</small>` : ''}
                    ` : '-'}
                </td>
                <td><strong>${student.total_amount.toFixed(2)} ₸</strong></td>
                <td>
                    ${student.remaining_debt > 0 ? `
                        <span style="color: #dc3545; font-weight: bold;">${student.remaining_debt.toFixed(2)} ₸</span>
                    ` : '<span style="color: #28a745;">0 ₸</span>'}
                </td>
                <td>${student.payment_percent.toFixed(1)}%</td>
                <td><span class="status-badge ${statusClass}">${status}</span></td>
                <td>
                    <button class="btn btn-primary btn-sm" onclick="openApplyPaymentModal('${student.full_name}', '${groupName}', ${yearId})" title="Применить оплату">
                        💰 Применить
                    </button>
                </td>
            `;
            tbody.appendChild(row);
        });
        
        reportContent.style.display = 'block';
        if (exportBtn) exportBtn.style.display = 'inline-block';
        
        showNotification('✅', 'Отчет загружен', 'success');
    } catch (error) {
        console.error('Error loading report:', error);
        showNotification('❌', 'Ошибка загрузки отчета: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

// Открытие модального окна применения оплаты
function openApplyPaymentModal(fullName, group, yearId) {
    const modal = document.getElementById('apply-payment-modal');
    const fullnameInput = document.getElementById('payment-fullname');
    const groupInput = document.getElementById('payment-group');
    const yearIdInput = document.getElementById('payment-year-id');
    
    if (!modal || !fullnameInput || !groupInput || !yearIdInput) return;
    
    fullnameInput.value = fullName;
    groupInput.value = group;
    yearIdInput.value = yearId;
    
    document.getElementById('payment-amount').value = '';
    document.getElementById('payment-notes').value = '';
    
    modal.style.display = 'flex';
    modal.classList.add('show');
}

// Применение оплаты
async function handleApplyPayment(e) {
    e.preventDefault();
    
    const fullName = document.getElementById('payment-fullname').value;
    const group = document.getElementById('payment-group').value;
    const yearId = document.getElementById('payment-year-id').value;
    const amount = parseFloat(document.getElementById('payment-amount').value);
    const notes = document.getElementById('payment-notes').value;
    
    if (!fullName || !group || !yearId || amount <= 0) {
        showNotification('⚠️', 'Заполните все обязательные поля', 'error');
        return;
    }
    
    try {
        showNotification('⏳', 'Применение оплаты...', 'info');
        
        await apiCall('/admin/reports/payment', {
            method: 'POST',
            body: JSON.stringify({
                full_name: fullName,
                group: group,
                academic_year_id: parseInt(yearId),
                applied_amount: amount,
                notes: notes
            })
        });
        
        // Закрываем модальное окно
        const modal = document.getElementById('apply-payment-modal');
        if (modal) {
            modal.style.display = 'none';
            modal.classList.remove('show');
        }
        
        // Перезагружаем отчет
        await loadGroupReport();
        
        showNotification('✅', 'Оплата успешно применена!', 'success');
    } catch (error) {
        console.error('Error applying payment:', error);
        showNotification('❌', 'Ошибка применения оплаты: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

// Экспорт отчета в Excel
async function exportGroupReport() {
    if (!currentReportData) {
        showNotification('⚠️', 'Сначала загрузите отчет', 'error');
        return;
    }
    
    const yearSelect = document.getElementById('report-year-select');
    const groupSelect = document.getElementById('report-group-select');
    
    if (!yearSelect || !groupSelect) return;
    
    const yearId = yearSelect.value;
    const groupName = groupSelect.value;
    
    try {
        showNotification('⏳', 'Экспорт отчета...', 'info');
        
        const response = await fetch(`/api/admin/reports/group/export?group=${encodeURIComponent(groupName)}&academic_year_id=${yearId}`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `report_${groupName}_${yearId}.xlsx`;
        a.click();
        window.URL.revokeObjectURL(url);
        
        showNotification('✅', 'Отчет успешно экспортирован!', 'success');
    } catch (error) {
        console.error('Error exporting report:', error);
        showNotification('❌', 'Ошибка экспорта: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

// Делаем функции глобальными
window.openApplyPaymentModal = openApplyPaymentModal;

// Переключение табов в настройках
function switchToTab(tabName) {
    // Убираем активный класс со всех табов
    document.querySelectorAll('.settings-tab').forEach(tab => {
        tab.classList.remove('active');
    });
    document.querySelectorAll('.tab-pane').forEach(pane => {
        pane.classList.remove('active');
    });
    
    // Активируем выбранный таб
    const activeTab = document.querySelector(`.settings-tab[data-tab="${tabName}"]`);
    const activePane = document.getElementById(`tab-${tabName}`);
    
    if (activeTab) activeTab.classList.add('active');
    if (activePane) activePane.classList.add('active');
    
    // Загружаем данные для выбранного таба
    switch(tabName) {
        case 'users':
            loadUsers();
            break;
        case 'groups':
            loadGroupsForAdmin();
            break;
        case 'banks':
            loadBanksForAdmin();
            break;
        case 'academic-years':
            loadAcademicYearsForAdmin();
            loadGroupsForAmountForm();
            loadAcademicYearsForAmountForm();
            loadGroupRequiredAmounts(); // Загружаем все установленные суммы
            break;
        case 'import':
            // Импорт уже настроен через обработчики событий
            break;
    }
}

// ========== ИМПОРТ СТУДЕНТОВ ==========
let importData = [];

// Скачивание шаблона для импорта
async function downloadImportTemplate() {
    try {
        showNotification('⏳', 'Загрузка шаблона...', 'info');
        
        const response = await fetch('/api/admin/import/template', {
            headers: {
                'Authorization': `Bearer ${authToken}`,
            },
        });

        if (!response.ok) {
            throw new Error('Ошибка загрузки шаблона');
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'шаблон_импорта_студентов.xlsx';
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        
        showNotification('✅', 'Шаблон успешно загружен!', 'success');
    } catch (error) {
        console.error('Error downloading template:', error);
        showNotification('❌', 'Ошибка загрузки шаблона: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

async function previewImportFile(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = (e) => {
            try {
                const data = new Uint8Array(e.target.result);
                const workbook = XLSX.read(data, { type: 'array' });
                const firstSheet = workbook.Sheets[workbook.SheetNames[0]];
                const jsonData = XLSX.utils.sheet_to_json(firstSheet);
                
                // Ищем колонки ФИО, ГРУППА, ЦЕНА (регистронезависимо)
                const students = jsonData.map(row => {
                    let fullName = '';
                    let group = '';
                    let price = 0;
                    
                    // Ищем колонки в разных вариантах написания
                    for (const key in row) {
                        const lowerKey = key.toLowerCase().trim();
                        if (lowerKey === 'фио' || lowerKey === 'fio' || lowerKey === 'полное имя' || lowerKey === 'имя') {
                            fullName = String(row[key] || '').trim();
                        } else if (lowerKey === 'группа' || lowerKey === 'group') {
                            group = String(row[key] || '').trim();
                        } else if (lowerKey === 'цена' || lowerKey === 'price' || lowerKey === 'сумма' || lowerKey === 'amount') {
                            price = parseFloat(row[key]) || 0;
                        }
                    }
                    
                    return { fullName, group, price };
                }).filter(s => s.fullName && s.group); // Фильтруем пустые строки
                
                if (students.length === 0) {
                    reject(new Error('Не найдено данных для импорта. Убедитесь, что файл содержит колонки: ФИО, ГРУППА, ЦЕНА'));
                    return;
                }
                
                importData = students;
                
                // Показываем предпросмотр
                const preview = document.getElementById('import-preview');
                const tbody = document.getElementById('import-preview-tbody');
                if (preview && tbody) {
                    tbody.innerHTML = students.slice(0, 10).map(s => `
                        <tr>
                            <td>${s.fullName}</td>
                            <td>${s.group}</td>
                            <td>${s.price.toFixed(2)} ₸</td>
                        </tr>
                    `).join('');
                    if (students.length > 10) {
                        tbody.innerHTML += `<tr><td colspan="3" style="text-align: center; color: #666;">... и еще ${students.length - 10} записей</td></tr>`;
                    }
                    preview.style.display = 'block';
                }
                
                resolve(students);
            } catch (error) {
                reject(new Error('Ошибка парсинга Excel файла: ' + error.message));
            }
        };
        reader.onerror = () => reject(new Error('Ошибка чтения файла'));
        reader.readAsArrayBuffer(file);
    });
}

async function handleImportStudents() {
    if (importData.length === 0) {
        showNotification('⚠️', 'Нет данных для импорта', 'error');
        return;
    }
    
    try {
        showNotification('⏳', 'Импорт студентов...', 'info');
        
        // Получаем активный учебный год
        const activeYear = await apiCall('/academic-years/active');
        if (!activeYear || !activeYear.ID) {
            showNotification('❌', 'Не найден активный учебный год', 'error');
            return;
        }
        
        // Отправляем данные на сервер
        const result = await apiCall('/admin/import/students', {
            method: 'POST',
            body: JSON.stringify({
                students: importData,
                academic_year_id: activeYear.ID || activeYear.id || activeYear.Id
            }),
        });
        
        showNotification('✅', `Успешно импортировано ${result.imported || importData.length} студентов`, 'success');
        
        // Закрываем модальное окно
        const modal = document.getElementById('import-modal');
        if (modal) {
            modal.style.display = 'none';
            modal.classList.remove('show');
        }
        
        // Очищаем данные
        importData = [];
        document.getElementById('import-file').value = '';
        document.getElementById('import-file-name').style.display = 'none';
        document.getElementById('import-preview').style.display = 'none';
        document.getElementById('import-submit-btn').style.display = 'none';
        
        // Обновляем список квитанций
        loadReceipts();
    } catch (error) {
        console.error('Import error:', error);
        showNotification('❌', 'Ошибка импорта: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

// Перенос переплаты на новый учебный год
async function transferOverpayment(fullName, group, overpaidAmount) {
    try {
        // Загружаем учебные года
        const years = await apiCall('/academic-years');
        if (!years || years.length < 2) {
            showNotification('❌', 'Необходимо минимум 2 учебных года для переноса', 'error');
            return;
        }

        // Заполняем форму
        document.getElementById('transfer-full-name').value = fullName;
        document.getElementById('transfer-group').value = group;
        document.getElementById('transfer-amount').value = overpaidAmount;
        
        // Заполняем селекты учебных годов
        const fromYearSelect = document.getElementById('transfer-from-year');
        const toYearSelect = document.getElementById('transfer-to-year');
        
        fromYearSelect.innerHTML = '<option value="">Выберите год</option>';
        toYearSelect.innerHTML = '<option value="">Выберите год</option>';
        
        years.forEach(year => {
            const option1 = document.createElement('option');
            option1.value = year.ID;
            option1.textContent = year.Year;
            fromYearSelect.appendChild(option1);
            
            const option2 = document.createElement('option');
            option2.value = year.ID;
            option2.textContent = year.Year;
            toYearSelect.appendChild(option2);
        });

        // Показываем модальное окно
        const modal = document.getElementById('transfer-overpayment-modal');
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    } catch (error) {
        console.error('Transfer overpayment error:', error);
        showNotification('❌', 'Ошибка загрузки данных: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

// Делаем функцию глобальной
window.transferOverpayment = transferOverpayment;

// Обработчик отправки формы переноса переплаты
async function handleTransferOverpayment() {
    try {
        const fullName = document.getElementById('transfer-full-name').value;
        const group = document.getElementById('transfer-group').value;
        const amount = parseFloat(document.getElementById('transfer-amount').value);
        const fromYearID = parseInt(document.getElementById('transfer-from-year').value);
        const toYearID = parseInt(document.getElementById('transfer-to-year').value);
        const notes = document.getElementById('transfer-notes').value;

        if (!fullName || !group || !amount || amount <= 0) {
            showNotification('❌', 'Заполните все обязательные поля', 'error');
            return;
        }

        if (!fromYearID || !toYearID) {
            showNotification('❌', 'Выберите учебные года', 'error');
            return;
        }

        if (fromYearID === toYearID) {
            showNotification('❌', 'Учебные года должны быть разными', 'error');
            return;
        }

        await apiCall('/admin/overpayment/transfer', {
            method: 'POST',
            body: JSON.stringify({
                full_name: fullName,
                group: group,
                amount: amount,
                from_academic_year_id: fromYearID,
                to_academic_year_id: toYearID,
                notes: notes
            })
        });

        showNotification('✅', 'Переплата успешно перенесена', 'success');
        
        // Закрываем модальное окно
        const modal = document.getElementById('transfer-overpayment-modal');
        modal.style.display = 'none';
        modal.classList.remove('show');
        
        // Очищаем форму
        document.getElementById('transfer-overpayment-form').reset();
        
        // Обновляем список квитанций
        loadReceipts();
    } catch (error) {
        console.error('Transfer overpayment error:', error);
        showNotification('❌', 'Ошибка переноса переплаты: ' + (error.message || 'Неизвестная ошибка'), 'error');
    }
}

