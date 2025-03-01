document.addEventListener('DOMContentLoaded', () => {
    const progressBar = document.getElementById('progress');
    const statusText = document.getElementById('status');
    const detailsText = document.getElementById('details');
    const startButton = document.getElementById('startBtn');
    const tokenInput = document.getElementById('token');
    const usernameInput = document.getElementById('username');
    const passwordInput = document.getElementById('password');
    const hostInput = document.getElementById('host');
    const localPathInput = document.getElementById('localPath');
    const remotePathInput = document.getElementById('remotePath');
    let eventSource = null;
    let currentSessionId = null;

    // Helper function to convert FormData to ArrayBuffer with boundary
    async function formDataToArrayBuffer(formData, boundary) {
        const chunks = [];
        for (let [key, value] of formData.entries()) {
            chunks.push(
                `--${boundary}\r\n` +
                `Content-Disposition: form-data; name="${key}"\r\n\r\n` +
                `${value}\r\n`
            );
        }
        chunks.push(`--${boundary}--\r\n`);

        // Convert chunks to ArrayBuffer
        const blob = new Blob(chunks, { type: 'text/plain' });
        return await blob.arrayBuffer();
    }

    function formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
    }

    function updateProgress(current, total) {
        const percentage = (current / total) * 100;
        progressBar.style.width = `${percentage}%`;
        detailsText.textContent = `${formatBytes(current)} / ${formatBytes(total)}`;
    }

    function validateForm() {
        let isValid = true;
        const inputs = [tokenInput, usernameInput, hostInput, localPathInput, remotePathInput];
        
        inputs.forEach(input => {
            if (!input.value.trim()) {
                input.classList.add('error');
                isValid = false;
            } else {
                input.classList.remove('error');
            }
        });

        return isValid;
    }

    function startProgressMonitoring(sessionId) {
        if (eventSource) {
            eventSource.close();
        }

        // Create SSE connection with session ID
        eventSource = new EventSource(`http://localhost:1314/api/deploy/progress?session_id=${sessionId}`, {
            headers: {
                'Accept': 'text/event-stream'
            }
        });

        // Handle incoming messages
        eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data);
            
            switch (data.event) {
                case 'progress':
                    statusText.textContent = 'Uploading...';
                    updateProgress(data.data.current, data.data.total);
                    break;

                case 'complete':
                    statusText.textContent = 'Deployment completed!';
                    updateProgress(data.data.current, data.data.total);
                    eventSource.close();
                    startButton.disabled = false;
                    currentSessionId = null;
                    break;

                case 'error':
                    statusText.textContent = 'Deployment failed!';
                    eventSource.close();
                    startButton.disabled = false;
                    currentSessionId = null;
                    break;
            }
        };

        // Handle errors
        eventSource.onerror = (error) => {
            console.error('SSE Error:', error);
            statusText.textContent = 'Connection error!';
            eventSource.close();
            startButton.disabled = false;
            currentSessionId = null;
        };
    }

    async function startDeploy() {
        if (!validateForm()) {
            statusText.textContent = 'Please fill in all required fields';
            return;
        }

        if (eventSource) {
            eventSource.close();
        }

        startButton.disabled = true;
        progressBar.style.width = '0';
        statusText.textContent = 'Starting deployment...';
        detailsText.textContent = '';

        try {
            // Create FormData
            const formData = new FormData();
            formData.append('host_name', hostInput.value);
            formData.append('host_token', tokenInput.value);
            formData.append('username', usernameInput.value);
            formData.append('local_path', localPathInput.value);
            formData.append('remote_path', remotePathInput.value);
            if (passwordInput.value) {
                formData.append('password', passwordInput.value);
            }

            // Generate boundary
            const boundary = "----WebKitFormBoundary" + Math.random().toString(36).substring(2, 9);
            
            // Convert FormData to ArrayBuffer
            const arrayBufferBody = await formDataToArrayBuffer(formData, boundary);

            // Send POST request to initialize deployment
            const response = await fetch('http://localhost:1314/api/deploy', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${tokenInput.value}`,
                    'Content-Type': `multipart/form-data; boundary=${boundary}`,
                },
                body: arrayBufferBody
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            // Get session ID from response
            const result = await response.json();
            if (!result.session_id) {
                throw new Error('No session ID received from server');
            }

            currentSessionId = result.session_id;
            statusText.textContent = 'Deployment initialized, starting file transfer...';

            // Start monitoring progress with the received session ID
            startProgressMonitoring(currentSessionId);

        } catch (error) {
            console.error('Deploy Error:', error);
            statusText.textContent = 'Deployment failed: ' + error.message;
            startButton.disabled = false;
            currentSessionId = null;
        }
    }

    // Add input event listeners to remove error class on type
    [tokenInput, usernameInput, passwordInput, hostInput, localPathInput, remotePathInput].forEach(input => {
        input.addEventListener('input', () => {
            input.classList.remove('error');
            if (statusText.textContent === 'Please fill in all required fields') {
                statusText.textContent = 'Ready to deploy';
            }
        });
    });

    startButton.addEventListener('click', startDeploy);
}); 