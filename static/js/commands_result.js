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

function addResultToTable(file, success, timestamp, logFile) {
    const resultsBody = document.getElementById('resultsBody');
    const row = document.createElement('tr');
    row.className = 'history-item';
    
    const timeStr = new Date(timestamp).toLocaleString();
    
    row.innerHTML = `
        <td>${file}</td>
        <td>
            <span class="status-badge ${success ? 'status-success' : 'status-failed'}">
                ${success ? 'Success' : 'Failed'}
            </span>
        </td>
        <td>${timeStr}</td>
        <td>
            <button class="btn btn-sm btn-outline-primary view-log-btn" data-log="${logFile}">
                View Log
            </button>
        </td>
    `;
    
    resultsBody.insertBefore(row, resultsBody.firstChild);
    
    row.querySelector('.view-log-btn').addEventListener('click', function() {
        viewLog(this.getAttribute('data-log'));
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
            
            addResultToTable(
                file, 
                result.success, 
                startTime, 
                result.log_file
            );
            
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

async function loadHistory() {
    try {
        const response = await fetch('/history');
        const history = await response.json();
        const historyBody = document.getElementById('historyBody');
        
        historyBody.innerHTML = '';
        
        if (history.length === 0) {
            historyBody.innerHTML = `<tr><td colspan="4" class="text-center">No history available</td></tr>`;
            return;
        }
        
        history.forEach(item => {
            const row = document.createElement('tr');
            row.className = 'history-item';
            
            const timestamp = new Date(item.timestamp);
            const timeStr = timestamp.toLocaleString();
            
            row.innerHTML = `
                <td>${item.filename}</td>
                <td>
                    <span class="status-badge ${item.success ? 'status-success' : 'status-failed'}">
                        ${item.success ? 'Success' : 'Failed'}
                    </span>
                </td>
                <td>${timeStr}</td>
                <td>
                    <button class="btn btn-sm btn-outline-primary view-log-btn" data-log="${item.output_path}">
                        View Log
                    </button>
                </td>
            `;
            
            historyBody.appendChild(row);
            
            row.querySelector('.view-log-btn').addEventListener('click', function() {
                viewLog(this.getAttribute('data-log'));
            });
        });
        
        const historyModal = new bootstrap.Modal(document.getElementById('historyModal'));
        historyModal.show();
        
    } catch (error) {
        showOutput(`Error loading history: ${error.message}`);
    }
}

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
        document.getElementById('historyBtn').addEventListener('click', loadHistory);
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
