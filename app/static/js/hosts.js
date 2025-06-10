async function loadHosts() {
    try {
        const response = await fetch('/hosts/list');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const hosts = await response.json();
        const hostsBody = document.getElementById('hostsTableBody');

        hostsBody.innerHTML = '';

        if (hosts.length === 0) {
            hostsBody.innerHTML = `<tr><td colspan="5" class="text-center">No hosts available</td></tr>`;
            return;
        }

        hosts.forEach(host => {
            const row = document.createElement('tr');
            
            // Форматируем время последней проверки
            let lastChecked = 'Never';
            if (host.last_checked) {
                const date = new Date(host.last_checked);
                lastChecked = date.toLocaleString();
            }

            // Добавляем иконку статуса
            const statusIcon = host.status === 'active' ? '🟢' : '🔴';
            
            row.innerHTML = `
                <td>${host.ip_address}</td>
                <td>${host.name}</td>
                <td>${statusIcon} ${host.status}</td>
                <td>${lastChecked}</td>
                <td>
                    <button class="btn btn-sm btn-outline-danger delete-host-btn" data-id="${host.id}">
                        Delete
                    </button>
                </td>
            `;

            hostsBody.appendChild(row);

            row.querySelector('.delete-host-btn').addEventListener('click', function() {
                deleteHost(this.getAttribute('data-id'));
            });
        });

    } catch (error) {
        console.error('Error loading hosts:', error);
        const hostsBody = document.getElementById('hostsTableBody');
        hostsBody.innerHTML = `
            <tr>
                <td colspan="5" class="text-center text-danger">
                    Error loading hosts: ${error.message}
                </td>
            </tr>
        `;
    }
}

let formSubmitted = false; // Флаг для отслеживания отправки

async function addHost() {
    const ipAddress = document.getElementById('hostIP').value;
    const name = document.getElementById('hostName').value;
    const submitBtn = document.getElementById('addHostBtn');
    
    // Блокируем кнопку
    submitBtn.disabled = true;
    submitBtn.textContent = "Adding...";
    
    try {
        const response = await fetch('/hosts/add', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ip_address: ipAddress, name: name }),
        });

        if (!response.ok) {
            const errorText = await response.text();
            alert(`Error: ${errorText}`);
            return;
        }
        
        document.getElementById('addHostForm').reset();
        loadHosts();
    } catch (error) {
        console.error('Error adding host:', error);
        alert('Error adding host');
    } finally {
        // Разблокируем кнопку
        submitBtn.disabled = false;
        submitBtn.textContent = "Add Host";
    }
}

window.onload = function() {
    loadHosts();
    
    // Обработчик с защитой от повторной привязки
    const form = document.getElementById('addHostForm');
    form.addEventListener('submit', function(e) {
        e.preventDefault();
        addHost();
    });
    
    // Дополнительно: предотвращаем дублирующую привязку
    if (!window.hostsInitialized) {
        window.hostsInitialized = true;
    } else {
        console.warn('Hosts script initialized multiple times!');
    }
};

async function deleteHost(id) {
    try {
        const response = await fetch(`/hosts/delete?id=${id}`, {
            method: 'DELETE',
        });

        if (response.ok) {
            loadHosts();
        } else {
            console.error('Error deleting host:', await response.text());
        }
    } catch (error) {
        console.error('Error deleting host:', error);
    }
}

// document.getElementById('addHostForm').addEventListener('submit', function(e) {
//     e.preventDefault();
//     addHost();
// });



window.onload = function() {
    loadHosts();
    document.getElementById('addHostForm').addEventListener('submit', function(e) {
        e.preventDefault();
        addHost();
    });
};
