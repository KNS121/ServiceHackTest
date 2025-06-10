let selectedFiles = [];
let runResults = [];

async function loadBatFiles() {
    try {
        const response = await fetch('/list');
        const data = await response.text();
        return data.split('|');
    } catch (error) {
        console.error('Error loading bat files:', error);
        showOutput(`Error loading batch files: ${error.message}`);
        return [];
    }
}

function createCard(file) {
    const col = document.createElement('div');
    col.className = 'col-md-4 mb-3';
    
    col.innerHTML = `
        <div class="card shadow-sm h-100">
            <div class="card-body d-flex align-items-center">
                <div class="form-check flex-grow-1">
                    <input class="form-check-input" type="checkbox" id="check-${file}" value="${file}">
                    <label class="form-check-label ms-2" for="check-${file}">${file}</label>
                </div>
            </div>
        </div>
    `;
    
    const checkbox = col.querySelector('.form-check-input');
    const card = col.querySelector('.card');
    
    card.addEventListener('click', (e) => {
        if (e.target !== checkbox) {
            checkbox.checked = !checkbox.checked;
            checkbox.dispatchEvent(new Event('change'));
        }
    });
    
    checkbox.addEventListener('change', function() {
        if (this.checked) {
            selectedFiles.push(file);
            card.classList.add('selected');
        } else {
            selectedFiles = selectedFiles.filter(f => f !== file);
            card.classList.remove('selected');
        }
        updateRunButton();
    });
    
    return col;
}

function updateRunButton() {
    const btn = document.getElementById('runSelectedBtn');
    if (selectedFiles.length > 0) {
        btn.disabled = false;
        btn.textContent = `Run Selected (${selectedFiles.length})`;
    } else {
        btn.disabled = true;
        btn.textContent = 'Run Selected';
    }
}

function showOutput(message) {
    const outputEl = document.getElementById('output');
    outputEl.textContent += message + '\n';
    outputEl.scrollTop = outputEl.scrollHeight;
}

function clearOutput() {
    document.getElementById('output').textContent = '';
}

function addResultToTable(result) {
    const row = document.createElement('tr');
    row.className = 'history-item';
    
    const date = new Date(result.timestamp);
    const formattedDate = date.toLocaleString();
    
    const statusBadge = result.success ? 
        '<span class="status-badge status-success">Success</span>' : 
        '<span class="status-badge status-failed">Failed</span>';
    
    // Форматируем информацию о хосте
    let hostInfo = result.host;
    if (result.hostName) {
        hostInfo = `${result.hostName}<br><small>${result.hostIP}</small>`;
    }
    
    // Обновленная структура с колонкой Host
    row.innerHTML = `
        <td>${result.filename}</td>
        <td>${hostInfo}</td>
        <td>${statusBadge}</td>
        <td>${formattedDate}</td>
        <td>
            <button class="btn btn-sm btn-info view-log-btn" data-logfile="${result.logFile}">View Log</button>
        </td>
    `;
    
    document.getElementById('resultsBody').prepend(row);
    
    row.querySelector('.view-log-btn').addEventListener('click', function() {
        viewLog(this.getAttribute('data-logfile'));
    });
}

async function viewLog(logFile) {
    try {
        const response = await fetch(`/result?file=${encodeURIComponent(logFile)}`);
        const logContent = await response.text();
        
        document.getElementById('logContent').textContent = logContent;
        const logModal = new bootstrap.Modal(document.getElementById('logModal'));
        logModal.show();
    } catch (error) {
        showOutput(`Error loading log: ${error.message}`);
    }
}

async function runSelected() {

    const hostSelect = document.getElementById('hostSelect');
    const selectedOption = hostSelect.options[hostSelect.selectedIndex];
    const selectedHost = hostSelect.value;

    if (selectedOption.disabled) {
        showOutput(`Cannot run on inactive host: ${selectedHost}`);
        return;
    }

    if (selectedFiles.length === 0) {
        showOutput("No files selected!");
        return;
    }
    
    const outputEl = document.getElementById('output');
    const progressBar = document.getElementById('progressBar');
    
    outputEl.textContent = `Running ${selectedFiles.length} files...\n`;
    progressBar.style.width = '0%';
    progressBar.textContent = '0%';
    
    let completed = 0;
    const total = selectedFiles.length;
    
    for (const file of selectedFiles) {
        try {
            showOutput(`--- Starting: ${file} ---`);
            
            const startTime = new Date();
            const response = await fetch(
            `/run?file=${encodeURIComponent(file)}&host=${encodeURIComponent(selectedHost)}`)
            const result = await response.json();
            
            showOutput(result.output);
            showOutput(`--- Completed: ${file} (${result.success ? 'Success' : 'Failed'}) ---`);
            
            addResultToTable({
                filename: file,
                success: result.success,
                timestamp: startTime,
                logFile: result.log_file,
                host: result.host, // Информация о хосте из сервера
                hostName: "", // Для новых результатов имя будет пустым
                hostIP: result.host // Используем host как IP
            });
            
            completed++;
            const percent = Math.round((completed / total) * 100);
            progressBar.style.width = `${percent}%`;
            progressBar.textContent = `${percent}%`;
            
        } catch (error) {
            showOutput(`Error running ${file}: ${error.message}`);
            addResultToTable(file, false, new Date(), 'error.log');
        }
    }
    
    showOutput(`\nCompleted all ${selectedFiles.length} files!`);
    selectedFiles = [];
    updateRunButton();
    
    document.querySelectorAll('.form-check-input').forEach(checkbox => {
        checkbox.checked = false;
        checkbox.closest('.card').classList.remove('selected');
    });
}

// function loadHistory() {
//     fetch('/history')
//         .then(response => response.json())
//         .then(history => {
//             const historyBody = document.getElementById('historyBody');
//             historyBody.innerHTML = '';
            
//             history.forEach(item => {
//                 const row = document.createElement('tr');
//                 const date = new Date(item.Timestamp);
//                 const formattedDate = date.toLocaleString();
//                 const statusBadge = item.Success ? 
//                     '<span class="status-badge status-success">Success</span>' : 
//                     '<span class="status-badge status-failed">Failed</span>';
                
//                 // Форматируем информацию о хосте
//                 let hostInfo = item.HostIP;
//                 if (item.HostName) {
//                     hostInfo = `${item.HostName}<br><small>${item.HostIP}</small>`;
//                 }
                
//                 row.innerHTML = `
//                     <td>${item.Filename}</td>
//                     <td>${hostInfo}</td>
//                     <td>${statusBadge}</td>
//                     <td>${formattedDate}</td>
//                     <td>
//                         <button class="btn btn-sm btn-info view-log-btn" data-logfile="${item.Output}">View Log</button>
//                     </td>
//                 `;
//                 historyBody.appendChild(row);
//             });
            
//             // Добавляем обработчики для кнопок просмотра лога
//             document.querySelectorAll('#historyBody .view-log-btn').forEach(btn => {
//                 btn.addEventListener('click', function() {
//                     const logFile = this.getAttribute('data-logfile');
//                     viewLog(logFile);
//                 });
//             });
//         });
// }

async function init() {
    // Загрузка хостов при инициализации
    await loadHosts();
    setInterval(loadHosts, 30000);

    try {
        const files = await loadBatFiles();
        const container = document.getElementById('batContainer');
        
        files.forEach(file => {
            container.appendChild(createCard(file));
        });
        
        document.getElementById('selectAllBtn').addEventListener('click', function() {
            selectedFiles = [];
            
            document.querySelectorAll('.form-check-input').forEach(checkbox => {
                checkbox.checked = true;
                selectedFiles.push(checkbox.value);
                checkbox.closest('.card').classList.add('selected');
            });
            
            updateRunButton();
        });
        
        document.getElementById('runSelectedBtn').addEventListener('click', runSelected);
        //document.getElementById('historyBtn').addEventListener('click', loadHistory);
        document.getElementById('clearOutputBtn').addEventListener('click', clearOutput);
        
        updateRunButton();
        
    } catch (error) {
        showOutput(`Initialization error: ${error.message}`);
    }
}


async function loadHosts() {
    try {
        const response = await fetch('/hosts/list');
        const hosts = await response.json();
        const hostSelect = document.getElementById('hostSelect');
        
        const currentValue = hostSelect.value;
        hostSelect.innerHTML = '';
        
        // Добавляем localhost (всегда активен)
        const localhostOption = document.createElement('option');
        localhostOption.value = "localhost";
        localhostOption.textContent = "localhost 🟢 Active";
        hostSelect.appendChild(localhostOption);
        
        // Добавляем хосты из базы
        hosts.forEach(host => {
            const option = document.createElement('option');
            option.value = host.ip_address;
            
            // Добавляем статус и блокируем неактивные
            const isActive = host.status === 'active';
            const status = isActive ? '🟢 Active' : '🔴 Inactive';
            option.textContent = `${host.name} (${host.ip_address}) - ${status}`;
            
            if (!isActive) {
                option.disabled = true;
            }
            
            if (host.ip_address === currentValue && isActive) {
                option.selected = true;
            }
            
            hostSelect.appendChild(option);
        });
        
        // Восстанавливаем выбор если нужно
        if (hostSelect.value !== currentValue) {
            hostSelect.value = "localhost";
        }
        
    } catch (error) {
        console.error('Error loading hosts:', error);
        showOutput(`Host load error: ${error.message}`);
    }
}

window.onload = init;
